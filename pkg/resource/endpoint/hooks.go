package endpoint

import (
	"errors"
	"fmt"
	"time"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/eventbridge"

	"github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
)

const (
	StatusActive       = "ACTIVE"
	StatusCreating     = "CREATING"
	StatusUpdating     = "UPDATING"
	StatusDeleting     = "DELETING"
	StatusCreateFailed = "CREATE_FAILED"
	StatusUpdateFailed = "UPDATE_FAILED"
	StatusDeleteFailed = "DELETE_FAILED"

	defaultRequeueDelay = time.Second * 5
)

var (
	requeueWaitWhileCreating = ackrequeue.NeededAfter(
		fmt.Errorf("endpoint in %q state, requeing", StatusCreating),
		defaultRequeueDelay,
	)

	requeueWaitWhileUpdating = ackrequeue.NeededAfter(
		fmt.Errorf("endpoint in %q state, cannot be modified or deleted", StatusUpdating),
		defaultRequeueDelay,
	)

	requeueWaitWhileDeleting = ackrequeue.NeededAfter(
		fmt.Errorf("endpoint in %q state, cannot be modified or deleted", StatusDeleting),
		defaultRequeueDelay,
	)
)

// TerminalStatuses are the status strings that are terminal states for an
// Endpoint
var TerminalStatuses = []string{
	StatusCreateFailed,
	StatusUpdateFailed,
	StatusDeleteFailed,
}

// TODO(@embano1): more validation needed?
func validateEndpointSpec(delta *ackcompare.Delta, spec v1alpha1.EndpointSpec) error {
	if err := validateEventBus(spec); err != nil {
		return err
	}

	if spec.RoutingConfig == nil || spec.RoutingConfig.FailoverConfig == nil {
		return fmt.Errorf("invalid Spec: %q must be set",
			"spec.routingConfig.failoverConfig", // currently only field in shape
		)
	}

	if delta != nil && delta.DifferentAt("Spec.RoleARN") {
		roleARN := spec.RoleARN
		if roleARN == nil || *roleARN == "" {
			return fmt.Errorf("invalid Spec: unsetting %q is not supported",
				"spec.roleARN", // currently only field in shape
			)
		}
	}

	return nil
}

func validateEventBus(spec v1alpha1.EndpointSpec) error {
	if len(spec.EventBuses) != 2 {
		return fmt.Errorf("invalid Spec: %q must contain exactly two event buses",
			"spec.eventBuses")
	}

	// event bus names must be identical
	arns := make([]string, 2)
	for i, b := range spec.EventBuses {
		if b.EventBusARN == nil {
			return fmt.Errorf("invalid Spec: %q[%d] event bus arn must be set",
				"spec.eventBuses",
				i,
			)
		}
		arnInfo, err := arn.Parse(*b.EventBusARN)
		if err != nil {
			return fmt.Errorf("invalid Spec: %q[%d] invalid arn: %w",
				"spec.eventBuses",
				i,
				err,
			)
		}
		arns[i] = arnInfo.Resource
	}

	if arns[0] != arns[1] {
		return fmt.Errorf("invalid Spec: %q event bus names must be identical",
			"spec.eventBuses",
		)
	}
	return nil
}

// endpointAvailable returns true if the supplied Endpoint is in an available
// status
func endpointAvailable(r *resource) bool {
	if r.ko.Status.State == nil {
		return false
	}
	state := *r.ko.Status.State
	return state == StatusActive
}

// endpointInMutatingState returns true if the supplied Endpoint is in the process of
// being created
func endpointInMutatingState(r *resource) bool {
	if r.ko.Status.State == nil {
		return false
	}
	state := *r.ko.Status.State
	return state == StatusCreating || state == StatusUpdating || state == StatusDeleting
}

// requeueWaitUntilCanModify returns a `ackrequeue.RequeueNeededAfter` struct
// explaining the Endpoint cannot be modified until it reaches an available
// status.
func requeueWaitUntilCanModify(r *resource) *ackrequeue.RequeueNeededAfter {
	if r.ko.Status.State == nil {
		return nil
	}
	status := *r.ko.Status.State
	msg := fmt.Sprintf(
		"Endpoint in '%s' state, cannot be modified.",
		status,
	)
	return ackrequeue.NeededAfter(
		errors.New(msg),
		defaultRequeueDelay,
	)
}

// if an optional desired field value is nil explicitly unset it in the request
// input
func unsetRemovedSpecFields(
	delta *ackcompare.Delta,
	spec v1alpha1.EndpointSpec,
	input *eventbridge.UpdateEndpointInput,
) {
	if delta.DifferentAt("Spec.Description") {
		if spec.Description == nil {
			input.SetDescription("")
		}
	}

	if delta.DifferentAt("Spec.ReplicationConfig") {
		if spec.ReplicationConfig == nil {
			input.SetReplicationConfig(&eventbridge.ReplicationConfig{State: aws.String("ENABLED")})
		}
	}
}

func customPreCompare(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	aDescr := a.ko.Spec.Description
	bDescr := b.ko.Spec.Description

	if !equalStrings(aDescr, bDescr) {
		delta.Add("Spec.Description", aDescr, bDescr)
	}

	aRole := a.ko.Spec.RoleARN
	bRole := b.ko.Spec.RoleARN

	if !equalStrings(aRole, bRole) {
		delta.Add("Spec.RoleARN", aRole, bRole)
	}

	aReplCfg := a.ko.Spec.ReplicationConfig
	bReplCfg := b.ko.Spec.ReplicationConfig

	if !equalReplicationConfigs(aReplCfg, bReplCfg) {
		delta.Add("Spec.ReplicationConfig", aReplCfg, bReplCfg)
	}
}

func equalStrings(a, b *string) bool {
	if a == nil {
		return b == nil || *b == ""
	}

	if a != nil && b == nil {
		return false
	}

	return (*a == "" && b == nil) || *a == *b
}

func equalReplicationConfigs(a, b *v1alpha1.ReplicationConfig) bool {
	// assumes API always returns replication config
	if (a == nil || a.State == nil || *a.State == "" || *a.State == "ENABLED") && *b.State == "ENABLED" {
		return true
	}

	if a != nil && a.State != nil && *a.State == "DISABLED" && *b.State == "DISABLED" {
		return true
	}

	return false
}
