package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eventbridge"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	ebv1alpha1 "github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
)

const (
	// TODO(embano1): hack to exclude kustomization files
	// naive approach excludes all yaml files starting with "k"
	filterPattern      = "[^k]*.yaml"
	controllerFilePath = "../../config/controller"
	rbacFilePath       = "../../config/rbac"
	controllerName     = "ack-eventbridge-controller"
	ackNamespace       = "ack-system"
)

var testBusName = envconf.RandomName("bus-test", 12)

func TestSuite(t *testing.T) {
	tags := []*ebv1alpha1.Tag{{
		Key:   aws.String("ack"),
		Value: aws.String("true"),
	}}

	tests := features.New("EventBridge Controller").
		Setup(createController()).
		Assess("controller is running", controllerRunning()).
		Assess("create event bus", createEventBus(tags)).
		Assess("event bus has synced", eventBusSynced(tags)).
		Feature()

	testEnv.Test(t, tests)
}

func createController() func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		r, err := resources.New(c.Client().RESTConfig())
		assert.NilError(t, err)

		awsEnvs := []corev1.EnvVar{
			{
				Name:  "AWS_REGION",
				Value: envCfg.Region,
			},
			{
				Name:  "AWS_ACCESS_KEY_ID",
				Value: envCfg.AccessKey,
			},
			{
				Name:  "AWS_SECRET_ACCESS_KEY",
				Value: envCfg.SecretKey,
			},
			{
				Name:  "AWS_SESSION_TOKEN",
				Value: envCfg.SessionToken,
			},
		}

		err = decoder.DecodeEachFile(
			ctx, os.DirFS(controllerFilePath), filterPattern,
			decoder.CreateHandler(r),
			mutateController(envCfg.CtrlImage, awsEnvs),
		)
		assert.NilError(t, err)

		err = decoder.DecodeEachFile(
			ctx, os.DirFS(rbacFilePath), filterPattern,
			decoder.CreateHandler(r),
		)
		assert.NilError(t, err)

		return ctx
	}
}

func controllerRunning() func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		client, err := c.NewClient()
		assert.NilError(t, err)

		var d appsv1.Deployment
		err = client.Resources().Get(ctx, controllerName, ackNamespace, &d)
		assert.NilError(t, err)

		readyCondition := conditions.New(client.Resources()).DeploymentConditionMatch(&d, appsv1.DeploymentAvailable, corev1.ConditionTrue)
		err = wait.For(readyCondition, wait.WithTimeout(time.Minute))
		assert.NilError(t, err)

		return ctx
	}
}

func createEventBus(tags []*ebv1alpha1.Tag) func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		r, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			t.Fail()
		}
		err = ebv1alpha1.AddToScheme(r.GetScheme())
		assert.NilError(t, err)

		r.WithNamespace(crNamespace)
		bus := eventBusFor(testBusName, crNamespace, tags...)
		err = r.Create(ctx, &bus)
		assert.NilError(t, err)

		return ctx
	}
}

func eventBusSynced(tags []*ebv1alpha1.Tag) func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		r, err := resources.New(c.Client().RESTConfig())
		assert.NilError(t, err)

		err = ebv1alpha1.AddToScheme(r.GetScheme())
		assert.NilError(t, err)

		var bus ebv1alpha1.EventBus
		r.WithNamespace(crNamespace)
		err = r.Get(ctx, testBusName, crNamespace, &bus)
		assert.NilError(t, err)

		syncedCondition := conditions.New(r).ResourceMatch(&bus, func(bus k8s.Object) bool {
			for _, cond := range bus.(*ebv1alpha1.EventBus).Status.Conditions {
				if cond.Type == ackv1alpha1.ConditionTypeResourceSynced && cond.Status == corev1.ConditionTrue {
					return true
				}
			}
			return false
		})

		err = wait.For(syncedCondition, wait.WithTimeout(time.Minute))
		assert.NilError(t, err)

		sdk := sdkClient(t)
		resp, err := sdk.DescribeEventBus(&eventbridge.DescribeEventBusInput{
			Name: aws.String(testBusName),
		})
		assert.NilError(t, err)
		assert.Equal(t, *resp.Name, testBusName, "compare bus: name mismatch")

		listResp, err := sdk.ListTagsForResourceWithContext(ctx, &eventbridge.ListTagsForResourceInput{
			ResourceARN: resp.Arn,
		})
		assert.NilError(t, err)

		tagMap := make(map[string]string)
		for _, tag := range listResp.Tags {
			tagMap[*tag.Key] = *tag.Value
		}

		for _, tag := range tags {
			v, ok := tagMap[*tag.Key]
			assert.Equal(t, true, ok, "compare tags: tag not found")
			assert.Equal(t, *tag.Value, v, "compare tags: tag value mismatch")
		}

		return ctx
	}
}
