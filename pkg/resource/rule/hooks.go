package rule

import (
	"context"
	"fmt"

	svcsdk "github.com/aws/aws-sdk-go/service/eventbridge"

	"github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
	svcapitypes "github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
)

func validateRuleSpec(spec v1alpha1.RuleSpec) error {
	if s := spec.State; s != nil {
		if !(*s == "ENABLED" || *s == "DISABLED") {
			return fmt.Errorf("invalid Spec: %q must be %q or %q",
				"spec.state",
				"ENABLED",
				"DISABLED",
			)
		}
	}

	emptyPattern := func() bool {
		return spec.EventPattern == nil || *spec.EventPattern == ""
	}

	emptySchedule := func() bool {
		return spec.ScheduleExpression == nil || *spec.ScheduleExpression == ""
	}

	if emptySchedule() && emptyPattern() {
		return fmt.Errorf("invalid Spec: at least one of %q or %q must be specified",
			"spec.eventPattern",
			"spec.scheduleExpression",
		)
	}
	return nil
}

// setResourceAdditionalFields will set the fields that are not returned by
// DescribeRule calls
func (rm *resourceManager) setResourceAdditionalFields(
	ctx context.Context,
	ko *svcapitypes.Rule,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.setResourceAdditionalFields")
	defer func() { exit(err) }()

	if ko.Status.ACKResourceMetadata != nil && ko.Status.ACKResourceMetadata.ARN != nil &&
		*ko.Status.ACKResourceMetadata.ARN != "" {
		// Set event data store tags
		ko.Spec.Tags, err = rm.getTags(ctx, string(*ko.Status.ACKResourceMetadata.ARN))
		if err != nil {
			return err
		}
	}

	//TODO Query targets

	return nil
}

// getTags retrieves a resource list of tags.
func (rm *resourceManager) getTags(
	ctx context.Context,
	resourceARN string,
) (tags []*svcapitypes.Tag, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.getTags")
	defer func() { exit(err) }()

	var listTagsResponse *svcsdk.ListTagsForResourceOutput
	listTagsResponse, err = rm.sdkapi.ListTagsForResourceWithContext(
		ctx,
		&svcsdk.ListTagsForResourceInput{
			ResourceARN: &resourceARN,
		},
	)
	rm.metrics.RecordAPICall("GET", "ListTagsForResource", err)
	if err != nil {
		return nil, err
	}
	for _, tag := range listTagsResponse.Tags {
		tags = append(tags, &svcapitypes.Tag{
			Key:   tag.Key,
			Value: tag.Value,
		})
	}
	return tags, nil
}

func (rm *resourceManager) preDeleteRule(ctx context.Context, r *resource) (latest *resource, err error) {
	listReq := svcsdk.ListTargetsByRuleInput{
		EventBusName: r.ko.Spec.EventBusName,
		Rule:         r.ko.Spec.Name,
	}
	targets, err := rm.sdkapi.ListTargetsByRuleWithContext(ctx, &listReq)
	if err != nil {
		return nil, err
	}

	var removeTargets []*string
	for _, t := range targets.Targets {
		cp := *t.Id
		removeTargets = append(removeTargets, &cp)
	}

	removeReq := svcsdk.RemoveTargetsInput{
		EventBusName: r.ko.Spec.EventBusName,
		Ids:          removeTargets,
		Rule:         r.ko.Spec.Name,
	}

	// ignoring response as partial failure would lead to reconcile with fresh input from list targets
	_, err = rm.sdkapi.RemoveTargetsWithContext(ctx, &removeReq)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (rm *resourceManager) putRuleTargets(ctx context.Context, r *resource) (latest *resource, err error) {
	putReq := svcsdk.PutTargetsInput{
		EventBusName: r.ko.Spec.EventBusName,
		Targets:      nil,
		Rule:         r.ko.Spec.Name,
	}
	_, err = rm.sdkapi.PutTargetsWithContext(ctx, &putReq)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
