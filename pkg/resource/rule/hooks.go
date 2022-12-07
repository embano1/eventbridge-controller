package rule

import (
	"context"
	"fmt"

	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
	svcapitypes "github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
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

		ko.Spec.Targets, err = rm.getTargets(ctx, *ko.Spec.Name, *ko.Spec.EventBusName)
		if err != nil {
			return err
		}
	}

	return nil
}

func equalStrings(a, b *string) bool {
	if a == nil {
		return b == nil || *b == ""
	}
	return (*a == "" && b == nil) || *a == *b
}

func equalZeroString(a *string) bool {
	return equalStrings(a, aws.String(""))
}

func customPreCompare(
	delta *ackcompare.Delta,
	a *resource,
	b *resource,
) {
	if len(a.ko.Spec.Tags) != len(b.ko.Spec.Tags) {
		delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
	} else if len(a.ko.Spec.Tags) > 0 {
		if !equalTags(a.ko.Spec.Tags, b.ko.Spec.Tags) {
			delta.Add("Spec.Tags", a.ko.Spec.Tags, b.ko.Spec.Tags)
		}
	}

	if len(a.ko.Spec.Targets) != len(b.ko.Spec.Targets) {
		delta.Add("Spec.Targets", a.ko.Spec.Targets, b.ko.Spec.Targets)
	} else if len(a.ko.Spec.Targets) > 0 {
		if !equalTargets(a.ko.Spec.Targets, b.ko.Spec.Targets) {
			delta.Add("Spec.Targets", a.ko.Spec.Targets, b.ko.Spec.Targets)
		}
	}
}
