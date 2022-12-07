package e2e

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	svcsdk "github.com/aws/aws-sdk-go/service/eventbridge"
	sqssvcsdk "github.com/aws/aws-sdk-go/service/sqs"
	"gotest.tools/v3/assert"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"

	eventbridge "github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
)

func mutateController(image string, env []corev1.EnvVar) decoder.DecodeOption {
	return decoder.MutateOption(func(obj k8s.Object) error {
		d, ok := obj.(*v1.Deployment)
		if !ok {
			// ignore non-deployment objects in input, e.g. ack-system namespace
			return nil
		}

		container := &d.Spec.Template.Spec.Containers[0]

		// only patch the ack controller in case of multiple deployments
		if d.Name != controllerName {
			return nil
		}

		/*// enables leader election
		// TODO: https://github.com/aws-controllers-k8s/community/issues/1578
		d.Spec.Replicas = aws.Int32(2)*/

		container.Image = image
		container.Command = []string{"/ko-app/controller"}
		container.Args = []string{
			"--enable-development-logging",
			"--log-level=debug",
			// "--enable-leader-election",
		}

		envVars := container.Env
		envVars = append(envVars, env...)
		container.Env = envVars
		return nil
	})
}

func eventBusFor(name, namespace string, tags ...*eventbridge.Tag) eventbridge.EventBus {
	return eventbridge.EventBus{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: eventbridge.EventBusSpec{
			Name: aws.String(name),
			Tags: tags,
		},
	}
}

func ruleFor(name, namespace, bus, pattern string, targets []*eventbridge.Target, tags ...*eventbridge.Tag) eventbridge.Rule {
	return eventbridge.Rule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: eventbridge.RuleSpec{
			EventBusRef: &v1alpha1.AWSResourceReferenceWrapper{
				From: &v1alpha1.AWSResourceReference{
					Name: aws.String(bus),
				},
			},
			EventPattern: aws.String(pattern),
			Name:         aws.String(name),
			Tags:         tags,
			Targets:      targets,
		},
	}
}

func ebSDKClient(t *testing.T) *svcsdk.EventBridge {
	s, err := session.NewSession(&aws.Config{
		Region: aws.String(envCfg.Region),
	})
	assert.NilError(t, err, "create eventbridge service client")

	return svcsdk.New(s)
}

func sqsSDKClient() (*sqssvcsdk.SQS, error) {
	s, err := session.NewSession(&aws.Config{
		Region: aws.String(envCfg.Region),
	})
	if err != nil {
		return nil, fmt.Errorf("create aws session: %w", err)
	}

	return sqssvcsdk.New(s), nil
}

func createSQSTestQueue(name string, tags []*eventbridge.Tag) env.Func {
	return func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
		sqssdk, err := sqsSDKClient()
		if err != nil {
			return ctx, fmt.Errorf("create sqs sdk client: %w", err)
		}

		sqstags := make(map[string]*string)
		for _, t := range tags {
			cp := *t
			sqstags[*cp.Key] = cp.Value
		}

		const sqsPolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "events.amazonaws.com"
      },
      "Resource": "arn:aws:sqs:*",
      "Action": [
        "sqs:GetQueueAttributes",
        "sqs:GetQueueUrl",
        "sqs:SendMessage"
      ]
    }
  ]
}`

		resp, err := sqssdk.CreateQueue(&sqssvcsdk.CreateQueueInput{
			QueueName: aws.String(name),
			Tags:      sqstags,
			Attributes: map[string]*string{
				"Policy": aws.String(sqsPolicy),
			},
		})
		if err != nil {
			return ctx, fmt.Errorf("create sqs queue: %w", err)
		}
		queueURL = *resp.QueueUrl
		klog.V(1).Infof("created test sqs queue %q", *resp.QueueUrl)

		const arnKey = "QueueArn"
		attrs, err := sqssdk.GetQueueAttributes(&sqssvcsdk.GetQueueAttributesInput{
			AttributeNames: []*string{aws.String(arnKey)},
			QueueUrl:       resp.QueueUrl,
		})
		if err != nil {
			return ctx, fmt.Errorf("get sqs queue attributes: %w", err)
		}

		if arn, ok := attrs.Attributes[arnKey]; !ok {
			return ctx, fmt.Errorf("get sqs queue attributes: value for %q not found", arnKey)
		} else {
			queueARN = *arn
		}

		return ctx, nil
	}
}

func destroySQSTestQueue() env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		sqssdk, err := sqsSDKClient()
		if err != nil {
			return ctx, fmt.Errorf("create sqs sdk client: %w", err)
		}

		_, err = sqssdk.DeleteQueue(&sqssvcsdk.DeleteQueueInput{
			QueueUrl: aws.String(queueURL),
		})
		if err != nil {
			return ctx, fmt.Errorf("destroy sqs queue: %w", err)
		}

		klog.V(1).Infof("destroyed test sqs queue %q", queueURL)
		return ctx, nil
	}
}
