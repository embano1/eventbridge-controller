package rule

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"

	svcapitypes "github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
)

func Test_validateTargets(t *testing.T) {
	tests := []struct {
		name    string
		targets []svcapitypes.Target
		wantErr string
	}{
		{
			name:    "empty list of targets",
			targets: nil,
			wantErr: "",
		}, {
			name: "two targets, one without id",
			targets: []svcapitypes.Target{
				{
					ARN: aws.String("arn:1"),
					ID:  nil,
				}, {
					ARN: aws.String("arn:2"),
					ID:  aws.String("id2"),
				},
			},
			wantErr: "invalid target: target ID and ARN must be specified",
		}, {
			name: "two targets, one without arn",
			targets: []svcapitypes.Target{
				{
					ARN: aws.String("arn:1"),
					ID:  aws.String("id1"),
				}, {
					ARN: nil,
					ID:  aws.String("id2"),
				},
			},
			wantErr: "invalid target: target ID and ARN must be specified",
		}, {
			name: "two targets, duplicate ids",
			targets: []svcapitypes.Target{
				{
					ARN: aws.String("arn:1"),
					ID:  aws.String("id1"),
				}, {
					ARN: aws.String("arn:2"),
					ID:  aws.String("id1"),
				},
			},
			wantErr: "invalid target: unique target ID is already used",
		}, {
			name: "two valid targets, different ids same arn",
			targets: []svcapitypes.Target{
				{
					ARN: aws.String("arn:1"),
					ID:  aws.String("id1"),
				}, {
					ARN: aws.String("arn:1"),
					ID:  aws.String("id2"),
				},
			},
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateTargets(tt.targets); tt.wantErr != "" {
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
