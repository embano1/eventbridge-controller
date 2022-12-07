package rule

import (
	"context"

	"github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws-controllers-k8s/runtime/pkg/util"
	"github.com/aws/aws-sdk-go/service/eventbridge"

	"github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
)

// getTags retrieves a resource list of tags.
func (rm *resourceManager) getTags(
	ctx context.Context,
	resourceARN string,
) (tags []*v1alpha1.Tag, err error) {
	rlog := log.FromContext(ctx)
	exit := rlog.Trace("rm.getTags")
	defer func() { exit(err) }()

	var listTagsResponse *eventbridge.ListTagsForResourceOutput
	listTagsResponse, err = rm.sdkapi.ListTagsForResourceWithContext(
		ctx,
		&eventbridge.ListTagsForResourceInput{
			ResourceARN: &resourceARN,
		},
	)
	rm.metrics.RecordAPICall("GET", "ListTagsForResource", err)
	if err != nil {
		return nil, err
	}
	for _, tag := range listTagsResponse.Tags {
		tags = append(tags, &v1alpha1.Tag{
			Key:   tag.Key,
			Value: tag.Value,
		})
	}
	return tags, nil
}

// syncRuleTags updates event bus tags
func (rm *resourceManager) syncRuleTags(
	ctx context.Context,
	latest *resource,
	desired *resource,
) (err error) {
	rlog := log.FromContext(ctx)
	exit := rlog.Trace("rm.syncRuleTags")
	defer func(err error) { exit(err) }(err)

	added, removed := computeTagsDelta(desired.ko.Spec.Tags, latest.ko.Spec.Tags)

	if len(removed) > 0 {
		_, err = rm.sdkapi.UntagResourceWithContext(
			ctx,
			&eventbridge.UntagResourceInput{
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
			&eventbridge.TagResourceInput{
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
	desired []*v1alpha1.Tag,
	latest []*v1alpha1.Tag,
) (added, removed []*v1alpha1.Tag) {
	var visitedIndexes []string
mainLoop:
	for _, aElement := range desired {
		visitedIndexes = append(visitedIndexes, *aElement.Key)
		for _, bElement := range latest {
			if equalStrings(aElement.Key, bElement.Key) {
				if !equalStrings(aElement.Value, bElement.Value) {
					added = append(added, bElement)
				}
				continue mainLoop
			}
		}
		removed = append(removed, aElement)
	}
	for _, bElement := range latest {
		if !util.InStrings(*bElement.Key, visitedIndexes) {
			added = append(added, bElement)
		}
	}
	return added, removed
}

// sdkTagsFromResourceTags transforms a *svcapitypes.Tag array to a *svcsdk.Tag array.
func sdkTagsFromResourceTags(rTags []*v1alpha1.Tag) []*eventbridge.Tag {
	tags := make([]*eventbridge.Tag, len(rTags))
	for i := range rTags {
		tags[i] = &eventbridge.Tag{
			Key:   rTags[i].Key,
			Value: rTags[i].Value,
		}
	}
	return tags
}

// sdkTagStringsFromResourceTags transforms a *svcapitypes.Tag array to a string array.
func sdkTagStringsFromResourceTags(rTags []*v1alpha1.Tag) []*string {
	tags := make([]*string, len(rTags))
	for i := range rTags {
		tags[i] = rTags[i].Key
	}
	return tags
}

// equalTags returns true if two Tag arrays are equal regardless of the order
// of their elements.
func equalTags(
	a []*v1alpha1.Tag,
	b []*v1alpha1.Tag,
) bool {
	added, removed := computeTagsDelta(a, b)
	return len(added) == 0 && len(removed) == 0
}
