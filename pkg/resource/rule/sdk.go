// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Code generated by ack-generate. DO NOT EDIT.

package rule

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcompare "github.com/aws-controllers-k8s/runtime/pkg/compare"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	ackrequeue "github.com/aws-controllers-k8s/runtime/pkg/requeue"
	ackrtlog "github.com/aws-controllers-k8s/runtime/pkg/runtime/log"
	"github.com/aws/aws-sdk-go/aws"
	svcsdk "github.com/aws/aws-sdk-go/service/eventbridge"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcapitypes "github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
)

// Hack to avoid import errors during build...
var (
	_ = &metav1.Time{}
	_ = strings.ToLower("")
	_ = &aws.JSONValue{}
	_ = &svcsdk.EventBridge{}
	_ = &svcapitypes.Rule{}
	_ = ackv1alpha1.AWSAccountID("")
	_ = &ackerr.NotFound
	_ = &ackcondition.NotManagedMessage
	_ = &reflect.Value{}
	_ = fmt.Sprintf("")
	_ = &ackrequeue.NoRequeue{}
)

// sdkFind returns SDK-specific information about a supplied resource
func (rm *resourceManager) sdkFind(
	ctx context.Context,
	r *resource,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkFind")
	defer func() {
		exit(err)
	}()
	// If any required fields in the input shape are missing, AWS resource is
	// not created yet. Return NotFound here to indicate to callers that the
	// resource isn't yet created.
	if rm.requiredFieldsMissingFromReadOneInput(r) {
		return nil, ackerr.NotFound
	}

	input, err := rm.newDescribeRequestPayload(r)
	if err != nil {
		return nil, err
	}

	var resp *svcsdk.DescribeRuleOutput
	resp, err = rm.sdkapi.DescribeRuleWithContext(ctx, input)
	rm.metrics.RecordAPICall("READ_ONE", "DescribeRule", err)
	if err != nil {
		if awsErr, ok := ackerr.AWSError(err); ok && awsErr.Code() == "ResourceNotFoundException" {
			return nil, ackerr.NotFound
		}
		return nil, err
	}

	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := r.ko.DeepCopy()

	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if resp.Arn != nil {
		arn := ackv1alpha1.AWSResourceName(*resp.Arn)
		ko.Status.ACKResourceMetadata.ARN = &arn
	}
	if resp.Description != nil {
		ko.Spec.Description = resp.Description
	} else {
		ko.Spec.Description = nil
	}
	if resp.EventBusName != nil {
		ko.Spec.EventBusName = resp.EventBusName
	} else {
		ko.Spec.EventBusName = nil
	}
	if resp.EventPattern != nil {
		ko.Spec.EventPattern = resp.EventPattern
	} else {
		ko.Spec.EventPattern = nil
	}
	if resp.Name != nil {
		ko.Spec.Name = resp.Name
	} else {
		ko.Spec.Name = nil
	}
	if resp.RoleArn != nil {
		ko.Spec.RoleARN = resp.RoleArn
	} else {
		ko.Spec.RoleARN = nil
	}
	if resp.ScheduleExpression != nil {
		ko.Spec.ScheduleExpression = resp.ScheduleExpression
	} else {
		ko.Spec.ScheduleExpression = nil
	}
	if resp.State != nil {
		ko.Spec.State = resp.State
	} else {
		ko.Spec.State = nil
	}

	rm.setStatusDefaults(ko)
	if err := rm.setResourceAdditionalFields(ctx, ko); err != nil {
		return nil, err
	}
	return &resource{ko}, nil
}

// requiredFieldsMissingFromReadOneInput returns true if there are any fields
// for the ReadOne Input shape that are required but not present in the
// resource's Spec or Status
func (rm *resourceManager) requiredFieldsMissingFromReadOneInput(
	r *resource,
) bool {
	return r.ko.Spec.Name == nil

}

// newDescribeRequestPayload returns SDK-specific struct for the HTTP request
// payload of the Describe API call for the resource
func (rm *resourceManager) newDescribeRequestPayload(
	r *resource,
) (*svcsdk.DescribeRuleInput, error) {
	res := &svcsdk.DescribeRuleInput{}

	if r.ko.Spec.EventBusName != nil {
		res.SetEventBusName(*r.ko.Spec.EventBusName)
	}
	if r.ko.Spec.Name != nil {
		res.SetName(*r.ko.Spec.Name)
	}

	return res, nil
}

// sdkCreate creates the supplied resource in the backend AWS service API and
// returns a copy of the resource with resource fields (in both Spec and
// Status) filled in with values from the CREATE API operation's Output shape.
func (rm *resourceManager) sdkCreate(
	ctx context.Context,
	desired *resource,
) (created *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkCreate")
	defer func() {
		exit(err)
	}()

	if err = validateRuleSpec(desired.ko.Spec); err != nil {
		return nil, ackerr.NewTerminalError(err)
	}

	input, err := rm.newCreateRequestPayload(ctx, desired)
	if err != nil {
		return nil, err
	}

	var resp *svcsdk.PutRuleOutput
	_ = resp
	resp, err = rm.sdkapi.PutRuleWithContext(ctx, input)
	rm.metrics.RecordAPICall("CREATE", "PutRule", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if resp.RuleArn != nil {
		arn := ackv1alpha1.AWSResourceName(*resp.RuleArn)
		ko.Status.ACKResourceMetadata.ARN = &arn
	}

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// newCreateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Create API call for the resource
func (rm *resourceManager) newCreateRequestPayload(
	ctx context.Context,
	r *resource,
) (*svcsdk.PutRuleInput, error) {
	res := &svcsdk.PutRuleInput{}

	if r.ko.Spec.Description != nil {
		res.SetDescription(*r.ko.Spec.Description)
	}
	if r.ko.Spec.EventBusName != nil {
		res.SetEventBusName(*r.ko.Spec.EventBusName)
	}
	if r.ko.Spec.EventPattern != nil {
		res.SetEventPattern(*r.ko.Spec.EventPattern)
	}
	if r.ko.Spec.Name != nil {
		res.SetName(*r.ko.Spec.Name)
	}
	if r.ko.Spec.RoleARN != nil {
		res.SetRoleArn(*r.ko.Spec.RoleARN)
	}
	if r.ko.Spec.ScheduleExpression != nil {
		res.SetScheduleExpression(*r.ko.Spec.ScheduleExpression)
	}
	if r.ko.Spec.State != nil {
		res.SetState(*r.ko.Spec.State)
	}
	if r.ko.Spec.Tags != nil {
		f7 := []*svcsdk.Tag{}
		for _, f7iter := range r.ko.Spec.Tags {
			f7elem := &svcsdk.Tag{}
			if f7iter.Key != nil {
				f7elem.SetKey(*f7iter.Key)
			}
			if f7iter.Value != nil {
				f7elem.SetValue(*f7iter.Value)
			}
			f7 = append(f7, f7elem)
		}
		res.SetTags(f7)
	}

	return res, nil
}

// sdkUpdate patches the supplied resource in the backend AWS service API and
// returns a new resource with updated fields.
func (rm *resourceManager) sdkUpdate(
	ctx context.Context,
	desired *resource,
	latest *resource,
	delta *ackcompare.Delta,
) (updated *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkUpdate")
	defer func() {
		exit(err)
	}()
	if immutableFieldChanges := rm.getImmutableFieldChanges(delta); len(immutableFieldChanges) > 0 {
		msg := fmt.Sprintf("Immutable Spec fields have been modified: %s", strings.Join(immutableFieldChanges, ","))
		return nil, ackerr.NewTerminalError(fmt.Errorf(msg))
	}

	if err = validateRuleSpec(desired.ko.Spec); err != nil {
		return nil, ackerr.NewTerminalError(err)
	}

	input, err := rm.newUpdateRequestPayload(ctx, desired)
	if err != nil {
		return nil, err
	}

	var resp *svcsdk.PutRuleOutput
	_ = resp
	resp, err = rm.sdkapi.PutRuleWithContext(ctx, input)
	rm.metrics.RecordAPICall("UPDATE", "PutRule", err)
	if err != nil {
		return nil, err
	}
	// Merge in the information we read from the API call above to the copy of
	// the original Kubernetes object we passed to the function
	ko := desired.ko.DeepCopy()

	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if resp.RuleArn != nil {
		arn := ackv1alpha1.AWSResourceName(*resp.RuleArn)
		ko.Status.ACKResourceMetadata.ARN = &arn
	}

	rm.setStatusDefaults(ko)
	return &resource{ko}, nil
}

// newUpdateRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Update API call for the resource
func (rm *resourceManager) newUpdateRequestPayload(
	ctx context.Context,
	r *resource,
) (*svcsdk.PutRuleInput, error) {
	res := &svcsdk.PutRuleInput{}

	if r.ko.Spec.Description != nil {
		res.SetDescription(*r.ko.Spec.Description)
	}
	if r.ko.Spec.EventBusName != nil {
		res.SetEventBusName(*r.ko.Spec.EventBusName)
	}
	if r.ko.Spec.EventPattern != nil {
		res.SetEventPattern(*r.ko.Spec.EventPattern)
	}
	if r.ko.Spec.Name != nil {
		res.SetName(*r.ko.Spec.Name)
	}
	if r.ko.Spec.RoleARN != nil {
		res.SetRoleArn(*r.ko.Spec.RoleARN)
	}
	if r.ko.Spec.ScheduleExpression != nil {
		res.SetScheduleExpression(*r.ko.Spec.ScheduleExpression)
	}
	if r.ko.Spec.State != nil {
		res.SetState(*r.ko.Spec.State)
	}
	if r.ko.Spec.Tags != nil {
		f7 := []*svcsdk.Tag{}
		for _, f7iter := range r.ko.Spec.Tags {
			f7elem := &svcsdk.Tag{}
			if f7iter.Key != nil {
				f7elem.SetKey(*f7iter.Key)
			}
			if f7iter.Value != nil {
				f7elem.SetValue(*f7iter.Value)
			}
			f7 = append(f7, f7elem)
		}
		res.SetTags(f7)
	}

	return res, nil
}

// sdkDelete deletes the supplied resource in the backend AWS service API
func (rm *resourceManager) sdkDelete(
	ctx context.Context,
	r *resource,
) (latest *resource, err error) {
	rlog := ackrtlog.FromContext(ctx)
	exit := rlog.Trace("rm.sdkDelete")
	defer func() {
		exit(err)
	}()
	if err := rm.preDeleteRule(ctx, ko); err != nil {
		return nil, err
	}
	input, err := rm.newDeleteRequestPayload(r)
	if err != nil {
		return nil, err
	}
	var resp *svcsdk.DeleteRuleOutput
	_ = resp
	resp, err = rm.sdkapi.DeleteRuleWithContext(ctx, input)
	rm.metrics.RecordAPICall("DELETE", "DeleteRule", err)
	return nil, err
}

// newDeleteRequestPayload returns an SDK-specific struct for the HTTP request
// payload of the Delete API call for the resource
func (rm *resourceManager) newDeleteRequestPayload(
	r *resource,
) (*svcsdk.DeleteRuleInput, error) {
	res := &svcsdk.DeleteRuleInput{}

	if r.ko.Spec.EventBusName != nil {
		res.SetEventBusName(*r.ko.Spec.EventBusName)
	}
	if r.ko.Spec.Name != nil {
		res.SetName(*r.ko.Spec.Name)
	}

	return res, nil
}

// setStatusDefaults sets default properties into supplied custom resource
func (rm *resourceManager) setStatusDefaults(
	ko *svcapitypes.Rule,
) {
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if ko.Status.ACKResourceMetadata.Region == nil {
		ko.Status.ACKResourceMetadata.Region = &rm.awsRegion
	}
	if ko.Status.ACKResourceMetadata.OwnerAccountID == nil {
		ko.Status.ACKResourceMetadata.OwnerAccountID = &rm.awsAccountID
	}
	if ko.Status.Conditions == nil {
		ko.Status.Conditions = []*ackv1alpha1.Condition{}
	}
}

// updateConditions returns updated resource, true; if conditions were updated
// else it returns nil, false
func (rm *resourceManager) updateConditions(
	r *resource,
	onSuccess bool,
	err error,
) (*resource, bool) {
	ko := r.ko.DeepCopy()
	rm.setStatusDefaults(ko)

	// Terminal condition
	var terminalCondition *ackv1alpha1.Condition = nil
	var recoverableCondition *ackv1alpha1.Condition = nil
	var syncCondition *ackv1alpha1.Condition = nil
	for _, condition := range ko.Status.Conditions {
		if condition.Type == ackv1alpha1.ConditionTypeTerminal {
			terminalCondition = condition
		}
		if condition.Type == ackv1alpha1.ConditionTypeRecoverable {
			recoverableCondition = condition
		}
		if condition.Type == ackv1alpha1.ConditionTypeResourceSynced {
			syncCondition = condition
		}
	}
	var termError *ackerr.TerminalError
	if rm.terminalAWSError(err) || err == ackerr.SecretTypeNotSupported || err == ackerr.SecretNotFound || errors.As(err, &termError) {
		if terminalCondition == nil {
			terminalCondition = &ackv1alpha1.Condition{
				Type: ackv1alpha1.ConditionTypeTerminal,
			}
			ko.Status.Conditions = append(ko.Status.Conditions, terminalCondition)
		}
		var errorMessage = ""
		if err == ackerr.SecretTypeNotSupported || err == ackerr.SecretNotFound || errors.As(err, &termError) {
			errorMessage = err.Error()
		} else {
			awsErr, _ := ackerr.AWSError(err)
			errorMessage = awsErr.Error()
		}
		terminalCondition.Status = corev1.ConditionTrue
		terminalCondition.Message = &errorMessage
	} else {
		// Clear the terminal condition if no longer present
		if terminalCondition != nil {
			terminalCondition.Status = corev1.ConditionFalse
			terminalCondition.Message = nil
		}
		// Handling Recoverable Conditions
		if err != nil {
			if recoverableCondition == nil {
				// Add a new Condition containing a non-terminal error
				recoverableCondition = &ackv1alpha1.Condition{
					Type: ackv1alpha1.ConditionTypeRecoverable,
				}
				ko.Status.Conditions = append(ko.Status.Conditions, recoverableCondition)
			}
			recoverableCondition.Status = corev1.ConditionTrue
			awsErr, _ := ackerr.AWSError(err)
			errorMessage := err.Error()
			if awsErr != nil {
				errorMessage = awsErr.Error()
			}
			recoverableCondition.Message = &errorMessage
		} else if recoverableCondition != nil {
			recoverableCondition.Status = corev1.ConditionFalse
			recoverableCondition.Message = nil
		}
	}
	// Required to avoid the "declared but not used" error in the default case
	_ = syncCondition
	if terminalCondition != nil || recoverableCondition != nil || syncCondition != nil {
		return &resource{ko}, true // updated
	}
	return nil, false // not updated
}

// terminalAWSError returns awserr, true; if the supplied error is an aws Error type
// and if the exception indicates that it is a Terminal exception
// 'Terminal' exception are specified in generator configuration
func (rm *resourceManager) terminalAWSError(err error) bool {
	if err == nil {
		return false
	}
	awsErr, ok := ackerr.AWSError(err)
	if !ok {
		return false
	}
	switch awsErr.Code() {
	case "InvalidEventPatternException",
		"ManagedRuleException":
		return true
	default:
		return false
	}
}

// getImmutableFieldChanges returns list of immutable fields from the
func (rm *resourceManager) getImmutableFieldChanges(
	delta *ackcompare.Delta,
) []string {
	var fields []string
	if delta.DifferentAt("Spec.EventBusName") {
		fields = append(fields, "EventBusName")
	}
	if delta.DifferentAt("Spec.Name") {
		fields = append(fields, "Name")
	}

	return fields
}
func (rm *resourceManager) KRtoSDK(
	r *resource,
) ([]*svcsdk.Target, error) {
	// DescribeRuleInput
	var res []*svcsdk.Target
	//
	// {0xc000638900 { Targets Targets targets targets targets} Targets []*Target Target []*eventbridge.Target 0xc000972ea0 0xc00041b340 map[ARN:0xc000869980 BatchParameters:0xc000644780 DeadLetterConfig:0xc0006449c0 EcsParameters:0xc00057e9c0 HTTPParameters:0xc00057ee40 ID:0xc00057ef00 Input:0xc00057efc0 InputPath:0xc00057f080 InputTransformer:0xc00057f380 KinesisParameters:0xc00057f5c0 RedshiftDataParameters:0xc00057fbc0 RetryPolicy:0xc00057fec0 RoleARN:0xc0005e2000 RunCommandParameters:0xc0005e2600 SQSParameters:0xc0005e2d80 SageMakerPipelineParameters:0xc0005e2b40]}
	// Target

	for _, krTarget := range r.ko.Spec.Targets {
		t := &svcsdk.Target{}
		if krTarget.ARN != nil {
			t.SetArn(*krTarget.ARN)
		}
		if krTarget.BatchParameters != nil {
			tf1 := &svcsdk.BatchParameters{}
			if krTarget.BatchParameters.ArrayProperties != nil {
				tf1f0 := &svcsdk.BatchArrayProperties{}
				if krTarget.BatchParameters.ArrayProperties.Size != nil {
					tf1f0.SetSize(*krTarget.BatchParameters.ArrayProperties.Size)
				}
				tf1.SetArrayProperties(tf1f0)
			}
			if krTarget.BatchParameters.JobDefinition != nil {
				tf1.SetJobDefinition(*krTarget.BatchParameters.JobDefinition)
			}
			if krTarget.BatchParameters.JobName != nil {
				tf1.SetJobName(*krTarget.BatchParameters.JobName)
			}
			if krTarget.BatchParameters.RetryStrategy != nil {
				tf1f3 := &svcsdk.BatchRetryStrategy{}
				if krTarget.BatchParameters.RetryStrategy.Attempts != nil {
					tf1f3.SetAttempts(*krTarget.BatchParameters.RetryStrategy.Attempts)
				}
				tf1.SetRetryStrategy(tf1f3)
			}
			t.SetBatchParameters(tf1)
		}
		if krTarget.DeadLetterConfig != nil {
			tf2 := &svcsdk.DeadLetterConfig{}
			if krTarget.DeadLetterConfig.ARN != nil {
				tf2.SetArn(*krTarget.DeadLetterConfig.ARN)
			}
			t.SetDeadLetterConfig(tf2)
		}
		if krTarget.EcsParameters != nil {
			tf3 := &svcsdk.EcsParameters{}
			if krTarget.EcsParameters.CapacityProviderStrategy != nil {
				tf3f0 := []*svcsdk.CapacityProviderStrategyItem{}
				for _, tf3f0iter := range krTarget.EcsParameters.CapacityProviderStrategy {
					tf3f0elem := &svcsdk.CapacityProviderStrategyItem{}
					if tf3f0iter.Base != nil {
						tf3f0elem.SetBase(*tf3f0iter.Base)
					}
					if tf3f0iter.CapacityProvider != nil {
						tf3f0elem.SetCapacityProvider(*tf3f0iter.CapacityProvider)
					}
					if tf3f0iter.Weight != nil {
						tf3f0elem.SetWeight(*tf3f0iter.Weight)
					}
					tf3f0 = append(tf3f0, tf3f0elem)
				}
				tf3.SetCapacityProviderStrategy(tf3f0)
			}
			if krTarget.EcsParameters.EnableECSManagedTags != nil {
				tf3.SetEnableECSManagedTags(*krTarget.EcsParameters.EnableECSManagedTags)
			}
			if krTarget.EcsParameters.EnableExecuteCommand != nil {
				tf3.SetEnableExecuteCommand(*krTarget.EcsParameters.EnableExecuteCommand)
			}
			if krTarget.EcsParameters.Group != nil {
				tf3.SetGroup(*krTarget.EcsParameters.Group)
			}
			if krTarget.EcsParameters.LaunchType != nil {
				tf3.SetLaunchType(*krTarget.EcsParameters.LaunchType)
			}
			if krTarget.EcsParameters.NetworkConfiguration != nil {
				tf3f5 := &svcsdk.NetworkConfiguration{}
				if krTarget.EcsParameters.NetworkConfiguration.AWSvpcConfiguration != nil {
					tf3f5f0 := &svcsdk.AwsVpcConfiguration{}
					if krTarget.EcsParameters.NetworkConfiguration.AWSvpcConfiguration.AssignPublicIP != nil {
						tf3f5f0.SetAssignPublicIp(*krTarget.EcsParameters.NetworkConfiguration.AWSvpcConfiguration.AssignPublicIP)
					}
					if krTarget.EcsParameters.NetworkConfiguration.AWSvpcConfiguration.SecurityGroups != nil {
						tf3f5f0f1 := []*string{}
						for _, tf3f5f0f1iter := range krTarget.EcsParameters.NetworkConfiguration.AWSvpcConfiguration.SecurityGroups {
							var tf3f5f0f1elem string
							tf3f5f0f1elem = *tf3f5f0f1iter
							tf3f5f0f1 = append(tf3f5f0f1, &tf3f5f0f1elem)
						}
						tf3f5f0.SetSecurityGroups(tf3f5f0f1)
					}
					if krTarget.EcsParameters.NetworkConfiguration.AWSvpcConfiguration.Subnets != nil {
						tf3f5f0f2 := []*string{}
						for _, tf3f5f0f2iter := range krTarget.EcsParameters.NetworkConfiguration.AWSvpcConfiguration.Subnets {
							var tf3f5f0f2elem string
							tf3f5f0f2elem = *tf3f5f0f2iter
							tf3f5f0f2 = append(tf3f5f0f2, &tf3f5f0f2elem)
						}
						tf3f5f0.SetSubnets(tf3f5f0f2)
					}
					tf3f5.SetAwsvpcConfiguration(tf3f5f0)
				}
				tf3.SetNetworkConfiguration(tf3f5)
			}
			if krTarget.EcsParameters.PlacementConstraints != nil {
				tf3f6 := []*svcsdk.PlacementConstraint{}
				for _, tf3f6iter := range krTarget.EcsParameters.PlacementConstraints {
					tf3f6elem := &svcsdk.PlacementConstraint{}
					if tf3f6iter.Expression != nil {
						tf3f6elem.SetExpression(*tf3f6iter.Expression)
					}
					if tf3f6iter.Type != nil {
						tf3f6elem.SetType(*tf3f6iter.Type)
					}
					tf3f6 = append(tf3f6, tf3f6elem)
				}
				tf3.SetPlacementConstraints(tf3f6)
			}
			if krTarget.EcsParameters.PlacementStrategy != nil {
				tf3f7 := []*svcsdk.PlacementStrategy{}
				for _, tf3f7iter := range krTarget.EcsParameters.PlacementStrategy {
					tf3f7elem := &svcsdk.PlacementStrategy{}
					if tf3f7iter.Field != nil {
						tf3f7elem.SetField(*tf3f7iter.Field)
					}
					if tf3f7iter.Type != nil {
						tf3f7elem.SetType(*tf3f7iter.Type)
					}
					tf3f7 = append(tf3f7, tf3f7elem)
				}
				tf3.SetPlacementStrategy(tf3f7)
			}
			if krTarget.EcsParameters.PlatformVersion != nil {
				tf3.SetPlatformVersion(*krTarget.EcsParameters.PlatformVersion)
			}
			if krTarget.EcsParameters.PropagateTags != nil {
				tf3.SetPropagateTags(*krTarget.EcsParameters.PropagateTags)
			}
			if krTarget.EcsParameters.ReferenceID != nil {
				tf3.SetReferenceId(*krTarget.EcsParameters.ReferenceID)
			}
			if krTarget.EcsParameters.Tags != nil {
				tf3f11 := []*svcsdk.Tag{}
				for _, tf3f11iter := range krTarget.EcsParameters.Tags {
					tf3f11elem := &svcsdk.Tag{}
					if tf3f11iter.Key != nil {
						tf3f11elem.SetKey(*tf3f11iter.Key)
					}
					if tf3f11iter.Value != nil {
						tf3f11elem.SetValue(*tf3f11iter.Value)
					}
					tf3f11 = append(tf3f11, tf3f11elem)
				}
				tf3.SetTags(tf3f11)
			}
			if krTarget.EcsParameters.TaskCount != nil {
				tf3.SetTaskCount(*krTarget.EcsParameters.TaskCount)
			}
			if krTarget.EcsParameters.TaskDefinitionARN != nil {
				tf3.SetTaskDefinitionArn(*krTarget.EcsParameters.TaskDefinitionARN)
			}
			t.SetEcsParameters(tf3)
		}
		if krTarget.HTTPParameters != nil {
			tf4 := &svcsdk.HttpParameters{}
			if krTarget.HTTPParameters.HeaderParameters != nil {
				tf4f0 := map[string]*string{}
				for tf4f0key, tf4f0valiter := range krTarget.HTTPParameters.HeaderParameters {
					var tf4f0val string
					tf4f0val = *tf4f0valiter
					tf4f0[tf4f0key] = &tf4f0val
				}
				tf4.SetHeaderParameters(tf4f0)
			}
			if krTarget.HTTPParameters.PathParameterValues != nil {
				tf4f1 := []*string{}
				for _, tf4f1iter := range krTarget.HTTPParameters.PathParameterValues {
					var tf4f1elem string
					tf4f1elem = *tf4f1iter
					tf4f1 = append(tf4f1, &tf4f1elem)
				}
				tf4.SetPathParameterValues(tf4f1)
			}
			if krTarget.HTTPParameters.QueryStringParameters != nil {
				tf4f2 := map[string]*string{}
				for tf4f2key, tf4f2valiter := range krTarget.HTTPParameters.QueryStringParameters {
					var tf4f2val string
					tf4f2val = *tf4f2valiter
					tf4f2[tf4f2key] = &tf4f2val
				}
				tf4.SetQueryStringParameters(tf4f2)
			}
			t.SetHttpParameters(tf4)
		}
		if krTarget.ID != nil {
			t.SetId(*krTarget.ID)
		}
		if krTarget.Input != nil {
			t.SetInput(*krTarget.Input)
		}
		if krTarget.InputPath != nil {
			t.SetInputPath(*krTarget.InputPath)
		}
		if krTarget.InputTransformer != nil {
			tf8 := &svcsdk.InputTransformer{}
			if krTarget.InputTransformer.InputPathsMap != nil {
				tf8f0 := map[string]*string{}
				for tf8f0key, tf8f0valiter := range krTarget.InputTransformer.InputPathsMap {
					var tf8f0val string
					tf8f0val = *tf8f0valiter
					tf8f0[tf8f0key] = &tf8f0val
				}
				tf8.SetInputPathsMap(tf8f0)
			}
			if krTarget.InputTransformer.InputTemplate != nil {
				tf8.SetInputTemplate(*krTarget.InputTransformer.InputTemplate)
			}
			t.SetInputTransformer(tf8)
		}
		if krTarget.KinesisParameters != nil {
			tf9 := &svcsdk.KinesisParameters{}
			if krTarget.KinesisParameters.PartitionKeyPath != nil {
				tf9.SetPartitionKeyPath(*krTarget.KinesisParameters.PartitionKeyPath)
			}
			t.SetKinesisParameters(tf9)
		}
		if krTarget.RedshiftDataParameters != nil {
			tf10 := &svcsdk.RedshiftDataParameters{}
			if krTarget.RedshiftDataParameters.Database != nil {
				tf10.SetDatabase(*krTarget.RedshiftDataParameters.Database)
			}
			if krTarget.RedshiftDataParameters.DBUser != nil {
				tf10.SetDbUser(*krTarget.RedshiftDataParameters.DBUser)
			}
			if krTarget.RedshiftDataParameters.SecretManagerARN != nil {
				tf10.SetSecretManagerArn(*krTarget.RedshiftDataParameters.SecretManagerARN)
			}
			if krTarget.RedshiftDataParameters.Sql != nil {
				tf10.SetSql(*krTarget.RedshiftDataParameters.Sql)
			}
			if krTarget.RedshiftDataParameters.StatementName != nil {
				tf10.SetStatementName(*krTarget.RedshiftDataParameters.StatementName)
			}
			if krTarget.RedshiftDataParameters.WithEvent != nil {
				tf10.SetWithEvent(*krTarget.RedshiftDataParameters.WithEvent)
			}
			t.SetRedshiftDataParameters(tf10)
		}
		if krTarget.RetryPolicy != nil {
			tf11 := &svcsdk.RetryPolicy{}
			if krTarget.RetryPolicy.MaximumEventAgeInSeconds != nil {
				tf11.SetMaximumEventAgeInSeconds(*krTarget.RetryPolicy.MaximumEventAgeInSeconds)
			}
			if krTarget.RetryPolicy.MaximumRetryAttempts != nil {
				tf11.SetMaximumRetryAttempts(*krTarget.RetryPolicy.MaximumRetryAttempts)
			}
			t.SetRetryPolicy(tf11)
		}
		if krTarget.RoleARN != nil {
			t.SetRoleArn(*krTarget.RoleARN)
		}
		if krTarget.RunCommandParameters != nil {
			tf13 := &svcsdk.RunCommandParameters{}
			if krTarget.RunCommandParameters.RunCommandTargets != nil {
				tf13f0 := []*svcsdk.RunCommandTarget{}
				for _, tf13f0iter := range krTarget.RunCommandParameters.RunCommandTargets {
					tf13f0elem := &svcsdk.RunCommandTarget{}
					if tf13f0iter.Key != nil {
						tf13f0elem.SetKey(*tf13f0iter.Key)
					}
					if tf13f0iter.Values != nil {
						tf13f0elemf1 := []*string{}
						for _, tf13f0elemf1iter := range tf13f0iter.Values {
							var tf13f0elemf1elem string
							tf13f0elemf1elem = *tf13f0elemf1iter
							tf13f0elemf1 = append(tf13f0elemf1, &tf13f0elemf1elem)
						}
						tf13f0elem.SetValues(tf13f0elemf1)
					}
					tf13f0 = append(tf13f0, tf13f0elem)
				}
				tf13.SetRunCommandTargets(tf13f0)
			}
			t.SetRunCommandParameters(tf13)
		}
		if krTarget.SageMakerPipelineParameters != nil {
			tf14 := &svcsdk.SageMakerPipelineParameters{}
			if krTarget.SageMakerPipelineParameters.PipelineParameterList != nil {
				tf14f0 := []*svcsdk.SageMakerPipelineParameter{}
				for _, tf14f0iter := range krTarget.SageMakerPipelineParameters.PipelineParameterList {
					tf14f0elem := &svcsdk.SageMakerPipelineParameter{}
					if tf14f0iter.Name != nil {
						tf14f0elem.SetName(*tf14f0iter.Name)
					}
					if tf14f0iter.Value != nil {
						tf14f0elem.SetValue(*tf14f0iter.Value)
					}
					tf14f0 = append(tf14f0, tf14f0elem)
				}
				tf14.SetPipelineParameterList(tf14f0)
			}
			t.SetSageMakerPipelineParameters(tf14)
		}
		if krTarget.SQSParameters != nil {
			tf15 := &svcsdk.SqsParameters{}
			if krTarget.SQSParameters.MessageGroupID != nil {
				tf15.SetMessageGroupId(*krTarget.SQSParameters.MessageGroupID)
			}
			t.SetSqsParameters(tf15)
		}

		res = append(res, t)
	}
	return res, nil
}
