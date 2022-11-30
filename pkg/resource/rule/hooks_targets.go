package rule

import (
	"context"
	"errors"
	"reflect"

	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	ackutil "github.com/aws-controllers-k8s/runtime/pkg/util"
	svcsdk "github.com/aws/aws-sdk-go/service/eventbridge"

	svcapitypes "github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
)

// TODO(embano1): add more input validation
func validateTargets(targets []svcapitypes.Target) error {
	seen := make(map[string]bool)

	for _, t := range targets {
		if equalZeroString(t.ID) || equalZeroString(t.ARN) {
			return errors.New("invalid target: target ID and ARN must be specified")
		}

		if seen[*t.ID] {
			return errors.New("invalid target: unique target ID is already used")
		}

		seen[*t.ID] = true
	}

	return nil
}

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
				// NOTE(a-hilaly,embano1): we might need to force the removal, in some cases?
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

// computeTargetsDelta computes the delta between the specified targets and
// returns added and removed targets
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
