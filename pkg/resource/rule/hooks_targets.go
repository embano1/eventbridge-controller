package rule

import (
	"context"
	"reflect"

	svcapitypes "github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	ackutil "github.com/aws-controllers-k8s/runtime/pkg/util"
	svcsdk "github.com/aws/aws-sdk-go/service/eventbridge"
)

// syncRuleTargets updates event bus tags
func (rm *resourceManager) syncRuleTargets(
	ctx context.Context,
	desired *resource,
	latest *resource,
) (err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.syncRuleTargets")
	defer func(err error) { exit(err) }(err)

	added, removed := computeTargetsDelta(latest.ko.Spec.Targets, desired.ko.Spec.Targets)

	if len(removed) > 0 {
		_, err = rm.sdkapi.RemoveTargetsWithContext(
			ctx,
			&svcsdk.RemoveTargetsInput{
				//NOTE(a-hilaly,embano1): we might need to force the removal, in some cases?
				// thinking annotations... terminal conditions...
				Rule:         desired.ko.Spec.Name,
				EventBusName: desired.ko.Spec.EventBusName,
				Ids:          removed,
			})
		rm.metrics.RecordAPICall("UPDATE", "RemoveTargets", err)
		if err != nil {
			return err
		}
	}

	if len(added) > 0 {
		_, err = rm.sdkapi.PutTargetsWithContext(
			ctx,
			&svcsdk.PutTargetsInput{
				Rule:         desired.ko.Spec.Name,
				EventBusName: desired.ko.Spec.EventBusName,
				Targets:      sdkTargetsFromResource(desired),
			})
		rm.metrics.RecordAPICall("UPDATE", "PutTargets", err)
		if err != nil {
			return err
		}
	}
	return nil
}

// computeTargetsDelta .
func computeTargetsDelta(
	a []*svcapitypes.Target,
	b []*svcapitypes.Target,
) (added []*svcapitypes.Target, removed []*string) {
	var visitedIndexes []string
mainLoop:
	for _, aElement := range a {
		visitedIndexes = append(visitedIndexes, *aElement.ID)
		for _, bElement := range b {
			if equalStrings(aElement.ID, bElement.ID) {
				if !reflect.DeepEqual(aElement, bElement) {
					added = append(added, bElement)
				}
				continue mainLoop
			}
		}
		removed = append(removed, aElement.ID)
	}
	for _, bElement := range b {
		if !ackutil.InStrings(*bElement.ID, visitedIndexes) {
			added = append(added, bElement)
		}
	}
	return added, removed
}
