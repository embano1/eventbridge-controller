package event_bus

import (
	"context"
	"errors"
	"testing"

	"github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackmetrics "github.com/aws-controllers-k8s/runtime/pkg/metrics"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/eventbridge"
	"github.com/aws/aws-sdk-go/service/eventbridge/eventbridgeiface"
	"github.com/stretchr/testify/assert"

	svcapitypes "github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
)

var arn = v1alpha1.AWSResourceName("arn:some:bus")

type ebMockAPIClient struct {
	eventbridgeiface.EventBridgeAPI
	tagInput   *eventbridge.TagResourceInput
	untagInput *eventbridge.UntagResourceInput
	calls      int
	response   error
}

func (e *ebMockAPIClient) TagResourceWithContext(_ aws.Context, input *eventbridge.TagResourceInput, _ ...request.Option) (*eventbridge.TagResourceOutput, error) {
	e.calls++
	e.tagInput = input
	return nil, e.response
}

func (e *ebMockAPIClient) UntagResourceWithContext(_ aws.Context, input *eventbridge.UntagResourceInput, _ ...request.Option) (*eventbridge.UntagResourceOutput, error) {
	e.calls++
	e.untagInput = input
	return nil, e.response
}

func Test_resourceManager_syncEventBusTags(t *testing.T) {
	type args struct {
		latest  *resource
		desired *resource
	}
	tests := []struct {
		name           string
		args           args
		wantCalls      int
		wantTagInput   *eventbridge.TagResourceInput
		wantUntagInput *eventbridge.UntagResourceInput
		wantErr        error
	}{
		{
			name: "api call fails untag one",
			args: args{
				latest: &resource{getResource([]*svcapitypes.Tag{{
					Key:   aws.String("key-1"),
					Value: aws.String("value-1"),
				}}...)},
				desired: &resource{
					ko: &svcapitypes.EventBus{
						Spec: svcapitypes.EventBusSpec{
							Tags: nil,
						},
					},
				},
			},
			wantCalls:    1,
			wantTagInput: nil,
			wantUntagInput: &eventbridge.UntagResourceInput{
				ResourceARN: (*string)(&arn),
				TagKeys:     []*string{aws.String("key-1")},
			},
			wantErr: errors.New("call failed"),
		}, {
			name: "remove one tag",
			args: args{
				latest: &resource{getResource([]*svcapitypes.Tag{{
					Key:   aws.String("key-1"),
					Value: aws.String("value-1"),
				}}...)},
				desired: &resource{getResource()},
			},
			wantCalls:    1,
			wantTagInput: nil,
			wantUntagInput: &eventbridge.UntagResourceInput{
				ResourceARN: (*string)(&arn),
				TagKeys:     []*string{aws.String("key-1")},
			},
			wantErr: nil,
		}, {
			name: "add tag one",
			args: args{
				latest: &resource{getResource()},
				desired: &resource{getResource([]*svcapitypes.Tag{{
					Key:   aws.String("key-1"),
					Value: aws.String("value-1"),
				}}...)},
			},
			wantCalls: 1,
			wantTagInput: &eventbridge.TagResourceInput{
				ResourceARN: (*string)(&arn),
				Tags: []*eventbridge.Tag{{
					Key:   aws.String("key-1"),
					Value: aws.String("value-1"),
				}},
			},
			wantUntagInput: nil,
			wantErr:        nil,
		}, {
			name: "no changes",
			args: args{
				latest: &resource{getResource([]*svcapitypes.Tag{{
					Key:   aws.String("key-1"),
					Value: aws.String("value-1"),
				}}...)},
				desired: &resource{getResource([]*svcapitypes.Tag{{
					Key:   aws.String("key-1"),
					Value: aws.String("value-1"),
				}}...)},
			},
			wantCalls:      0,
			wantTagInput:   nil,
			wantUntagInput: nil,
			wantErr:        nil,
		}, {
			name: "two tags added, one remove",
			args: args{
				latest: &resource{getResource([]*svcapitypes.Tag{{
					Key:   aws.String("key-1"),
					Value: aws.String("value-1"),
				}}...)},
				desired: &resource{getResource([]*svcapitypes.Tag{
					{
						Key:   aws.String("key-2"),
						Value: aws.String("value-2"),
					},
					{
						Key:   aws.String("key-3"),
						Value: aws.String("value-3"),
					},
				}...)},
			},
			wantCalls: 2,
			wantTagInput: &eventbridge.TagResourceInput{
				ResourceARN: (*string)(&arn),
				Tags: []*eventbridge.Tag{
					{
						Key:   aws.String("key-2"),
						Value: aws.String("value-2"),
					}, {
						Key:   aws.String("key-3"),
						Value: aws.String("value-3"),
					},
				},
			},
			wantUntagInput: &eventbridge.UntagResourceInput{
				ResourceARN: (*string)(&arn),
				TagKeys:     []*string{aws.String("key-1")},
			},
			wantErr: nil,
		}, {
			name: "tags order changed, no api call needed",
			args: args{
				latest: &resource{getResource([]*svcapitypes.Tag{
					{
						Key:   aws.String("key-1"),
						Value: aws.String("value-1"),
					},
					{
						Key:   aws.String("key-2"),
						Value: aws.String("value-2"),
					},
				}...)},
				desired: &resource{getResource([]*svcapitypes.Tag{
					{
						Key:   aws.String("key-2"),
						Value: aws.String("value-2"),
					},
					{
						Key:   aws.String("key-1"),
						Value: aws.String("value-1"),
					},
				}...)},
			},
			wantCalls:      0,
			wantTagInput:   nil,
			wantUntagInput: nil,
			wantErr:        nil,
		}, {
			name: "one tag value changed",
			args: args{
				latest: &resource{getResource([]*svcapitypes.Tag{{
					Key:   aws.String("key-1"),
					Value: aws.String("value-1"),
				}}...)},
				desired: &resource{getResource([]*svcapitypes.Tag{{
					Key:   aws.String("key-1"),
					Value: aws.String("value-2"),
				}}...)},
			},
			wantCalls: 1,
			wantTagInput: &eventbridge.TagResourceInput{
				ResourceARN: (*string)(&arn),
				Tags: []*eventbridge.Tag{{
					Key:   aws.String("key-1"),
					Value: aws.String("value-2"),
				}},
			},
			wantUntagInput: nil,
			wantErr:        nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := ebMockAPIClient{
				response: tt.wantErr,
			}
			rm := &resourceManager{
				metrics: ackmetrics.NewMetrics("eventbridge"),
				sdkapi:  &api,
			}
			if err := rm.syncEventBusTags(context.TODO(), tt.args.latest, tt.args.desired); err != tt.wantErr {
				t.Errorf("syncEventBusTags() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantCalls, api.calls)
			assert.Equal(t, tt.wantTagInput, api.tagInput)
			assert.Equal(t, tt.wantUntagInput, api.untagInput)
		})
	}
}

func getResource(tags ...*svcapitypes.Tag) *svcapitypes.EventBus {
	return &svcapitypes.EventBus{
		Spec: svcapitypes.EventBusSpec{
			Tags: tags,
		},
		Status: svcapitypes.EventBusStatus{
			ACKResourceMetadata: &v1alpha1.ResourceMetadata{
				ARN: &arn,
			},
		},
	}
}
