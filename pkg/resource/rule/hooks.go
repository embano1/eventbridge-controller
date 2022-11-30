package rule

import (
	"context"
	"fmt"

	svcsdk "github.com/aws/aws-sdk-go/service/eventbridge"

	"github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
	svcapitypes "github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	ackutil "github.com/aws-controllers-k8s/runtime/pkg/util"
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
		Targets:      sdkTargetsFromResource(r),
		Rule:         r.ko.Spec.Name,
	}
	_, err = rm.sdkapi.PutTargetsWithContext(ctx, &putReq)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// syncRuleTags updates event bus tags
func (rm *resourceManager) syncRuleTags(
	ctx context.Context,
	latest *resource,
	desired *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncRuleTags")
	defer func(err error) { exit(err) }(err)

	added, removed := computeTagsDelta(latest.ko.Spec.Tags, desired.ko.Spec.Tags)

	if len(removed) > 0 {
		_, err = rm.sdkapi.UntagResourceWithContext(
			ctx,
			&svcsdk.UntagResourceInput{
				ResourceARN: (*string)(latest.ko.Status.ACKResourceMetadata.ARN),
				TagKeys:     sdkTagStringsFromResourceTags(removed),
			})

		rm.metrics.RecordAPICall("UPDATE", "RemoveTags", err)
		if err != nil {
			return err
		}
	}

	if len(added) > 0 {
		_, err = rm.sdkapi.TagResourceWithContext(
			ctx,
			&svcsdk.TagResourceInput{
				ResourceARN: (*string)(latest.ko.Status.ACKResourceMetadata.ARN),
				Tags:        sdkTagsFromResourceTags(added),
			})

		rm.metrics.RecordAPICall("UPDATE", "AddTags", err)
		if err != nil {
			return err
		}
	}
	return nil
}

// computeTagsDelta compares two Tag arrays and return two different lists
// containing the added and removed tags.
// The removed tags list only contains the Key of tags
func computeTagsDelta(
	a []*svcapitypes.Tag,
	b []*svcapitypes.Tag,
) (added, removed []*svcapitypes.Tag) {
	var visitedIndexes []string
mainLoop:
	for _, aElement := range a {
		visitedIndexes = append(visitedIndexes, *aElement.Key)
		for _, bElement := range b {
			if equalStrings(aElement.Key, bElement.Key) {
				if !equalStrings(aElement.Value, bElement.Value) {
					added = append(added, bElement)
				}
				continue mainLoop
			}
		}
		removed = append(removed, aElement)
	}
	for _, bElement := range b {
		if !ackutil.InStrings(*bElement.Key, visitedIndexes) {
			added = append(added, bElement)
		}
	}
	return added, removed
}

func equalStrings(a, b *string) bool {
	if a == nil {
		return b == nil || *b == ""
	}
	return (*a == "" && b == nil) || *a == *b
}

// sdkTagsFromResourceTags transforms a *svcapitypes.Tag array to a *svcsdk.Tag array.
func sdkTagsFromResourceTags(rTags []*svcapitypes.Tag) []*svcsdk.Tag {
	tags := make([]*svcsdk.Tag, len(rTags))
	for i := range rTags {
		tags[i] = &svcsdk.Tag{
			Key:   rTags[i].Key,
			Value: rTags[i].Value,
		}
	}
	return tags
}

// sdkTagStringsFromResourceTags transforms a *svcapitypes.Tag array to a string array.
func sdkTagStringsFromResourceTags(rTags []*svcapitypes.Tag) []*string {
	tags := make([]*string, len(rTags))
	for i := range rTags {
		tags[i] = rTags[i].Key
	}
	return tags
}
