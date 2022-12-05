package e2e

import (
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
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"

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

		container.Image = image
		container.Command = []string{"/ko-app/controller"}

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
