package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eventbridge"
	"github.com/aws/aws-sdk-go/service/sqs"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
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
	// naive approach excludes all yaml files starting with "k"
	// TODO(embano1): hack to exclude kustomization files
	filterPattern = "[^k]*.yaml"

	controllerFilePath = "../../config/controller"
	rbacFilePath       = "../../config/rbac"
	controllerName     = "ack-eventbridge-controller"
	ackNamespace       = "ack-system"

	testEventPattern = `{"detail-type": ["ack-e2e-testevent"]}`
)

var (
	testBusName  string
	testRuleName string
)

func TestSuite(t *testing.T) {
	testBusName = envconf.RandomName("ack-bus-e2e", 20)
	testRuleName = envconf.RandomName("ack-rule-e2e", 20)

	tests := features.New("EventBridge E2E").
		Setup(createController()).
		Assess("controller is running", controllerRunning()).
		Assess("create event bus", createEventBus(testBusName, testNamespace, tags)).
		Assess("event bus has synced", eventBusSynced(testBusName, testNamespace, tags)).
		Assess("create rule", createRule(testRuleName, testNamespace, testBusName, tags)).
		Assess("rule has synced", ruleSynced(testRuleName, testNamespace, testBusName, tags)).
		Assess("event received in sqs", eventReceived(testBusName)).
		Assess("delete test resources", deleteTestResources(testRuleName, testBusName, testNamespace)).
		Feature()

	testEnv.Test(t, tests)
}

func createController() func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		klog.V(1).Info("creating eventbridge controller")
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

func createEventBus(name, namespace string, tags []*ebv1alpha1.Tag) func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		r, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			t.Fail()
		}
		err = ebv1alpha1.AddToScheme(r.GetScheme())
		assert.NilError(t, err)

		r.WithNamespace(namespace)
		bus := eventBusFor(name, namespace, tags...)
		err = r.Create(ctx, &bus)
		assert.NilError(t, err)

		return ctx
	}
}

func eventBusSynced(name, namespace string, tags []*ebv1alpha1.Tag) func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		r, err := resources.New(c.Client().RESTConfig())
		assert.NilError(t, err)

		err = ebv1alpha1.AddToScheme(r.GetScheme())
		assert.NilError(t, err)

		var bus ebv1alpha1.EventBus
		r.WithNamespace(namespace)
		err = r.Get(ctx, name, namespace, &bus)
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

		sdk := ebSDKClient(t)
		resp, err := sdk.DescribeEventBus(&eventbridge.DescribeEventBusInput{
			Name: aws.String(name),
		})
		assert.NilError(t, err)
		assert.Equal(t, *resp.Name, name, "compare bus: name mismatch")

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

func createRule(name, namespace, bus string, tags []*ebv1alpha1.Tag) func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		r, err := resources.New(c.Client().RESTConfig())
		if err != nil {
			t.Fail()
		}
		err = ebv1alpha1.AddToScheme(r.GetScheme())
		assert.NilError(t, err)

		targets := []*ebv1alpha1.Target{{
			ARN: aws.String(queueARN),
			ID:  aws.String(queueName),
		}}

		r.WithNamespace(namespace)
		rule := ruleFor(name, namespace, bus, testEventPattern, targets, tags...)
		err = r.Create(ctx, &rule)
		assert.NilError(t, err)

		return ctx
	}
}

func ruleSynced(name, namespace, busName string, tags []*ebv1alpha1.Tag) func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		r, err := resources.New(c.Client().RESTConfig())
		assert.NilError(t, err)

		err = ebv1alpha1.AddToScheme(r.GetScheme())
		assert.NilError(t, err)

		var rule ebv1alpha1.Rule
		r.WithNamespace(namespace)
		err = r.Get(ctx, name, namespace, &rule)
		assert.NilError(t, err)

		syncedCondition := conditions.New(r).ResourceMatch(&rule, func(rule k8s.Object) bool {
			for _, cond := range rule.(*ebv1alpha1.Rule).Status.Conditions {
				if cond.Type == ackv1alpha1.ConditionTypeResourceSynced && cond.Status == corev1.ConditionTrue {
					return true
				}
			}
			return false
		})

		err = wait.For(syncedCondition, wait.WithTimeout(time.Minute))
		assert.NilError(t, err)

		sdk := ebSDKClient(t)
		resp, err := sdk.DescribeRule(&eventbridge.DescribeRuleInput{
			EventBusName: aws.String(busName),
			Name:         aws.String(name),
		})
		assert.NilError(t, err)
		assert.Equal(t, *resp.Name, name, "compare rule: name mismatch")

		targets, err := sdk.ListTargetsByRuleWithContext(ctx, &eventbridge.ListTargetsByRuleInput{
			EventBusName: aws.String(busName),
			Rule:         aws.String(name),
		})
		assert.NilError(t, err)
		assert.Equal(t, len(targets.Targets), len(rule.Spec.Targets), "compare rule targets: count mismatch")

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
			assert.Equal(t, true, ok, "compare tags: tag %q not found", *tag.Key)
			assert.Equal(t, *tag.Value, v, "compare tags: tag %q value mismatch", *tag.Key)
		}

		return ctx
	}
}

func eventReceived(busName string) func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		now := time.Now().UTC()
		payload := map[string]interface{}{
			"message":       "test event from ack e2e suite for eventbridge",
			"sentTimestamp": now,
		}

		payloadbytes, err := json.Marshal(payload)
		assert.NilError(t, err)

		testEvent := eventbridge.PutEventsInput{
			Entries: []*eventbridge.PutEventsRequestEntry{{
				Detail:       aws.String(string(payloadbytes)),
				DetailType:   aws.String("ack-e2e-testevent"),
				EventBusName: aws.String(busName),
				Resources:    []*string{&testNamespace, &testBusName, &testRuleName},
				Source:       aws.String("kubernetes.io/ack-e2e"),
				Time:         aws.Time(time.Now().UTC()),
			}},
		}

		receiveTimeout := time.Minute // rule pattern sync is eventually consistent
		timeoutctx, cancel := context.WithTimeout(ctx, receiveTimeout)
		defer cancel()

		// event sender
		go func() {
			ebsdk := ebSDKClient(t)
			ticker := time.NewTicker(time.Second * 5)
			defer ticker.Stop()

			attempts := 0
			for {
				attempts++

				select {
				case <-ticker.C:
					klog.V(1).Infof("sending test event: attempt %d", attempts)

					resp, err := ebsdk.PutEventsWithContext(ctx, &testEvent)
					assert.NilError(t, err)
					assert.Equal(t, *resp.FailedEntryCount, int64(0), "send test event: failed entry count is not 0")
				case <-timeoutctx.Done():
					return
				}
			}
		}()

		sqssdk, err := sqsSDKClient()
		assert.NilError(t, err)

		// event receiver
		var msgs []*sqs.Message
		received := func() (done bool, err error) {
			rcvResp, err := sqssdk.ReceiveMessageWithContext(ctx, &sqs.ReceiveMessageInput{
				AttributeNames:  []*string{aws.String("All")},
				QueueUrl:        aws.String(queueURL),
				WaitTimeSeconds: aws.Int64(3),
			})
			if err != nil {
				return false, fmt.Errorf("receive sqs message: %w", err)
			}

			if len(rcvResp.Messages) > 0 {
				klog.V(1).Infof("received new messages from sqs")
				msgs = rcvResp.Messages
				return true, nil
			}
			return false, nil
		}

		klog.V(1).Infof("waiting for messages from sqs")
		err = wait.For(received, wait.WithTimeout(receiveTimeout))
		assert.NilError(t, err)
		assert.Assert(t, len(msgs) > 0, "receive sqs message: no messages received")

		msgbody := msgs[0].Body
		assert.Assert(t, msgbody != nil, "receive sqs message: body is nil")

		var ebevent events.CloudWatchEvent
		err = json.Unmarshal([]byte(*msgbody), &ebevent)
		assert.NilError(t, err, "receive sqs message: unmarshal body")

		assert.Equal(
			t,
			string(ebevent.Detail),
			string(payloadbytes),
			"receive sqs message: compare send and receive payloads",
		)

		return ctx
	}
}

// assert that all eventbridge control plane resources are deleted
func deleteTestResources(ruleName, busName, namespace string) func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		r, err := resources.New(c.Client().RESTConfig())
		assert.NilError(t, err)

		// delete rule
		var rule ebv1alpha1.Rule
		err = r.Get(ctx, ruleName, namespace, &rule)
		assert.NilError(t, err)

		err = r.Delete(ctx, &rule)
		assert.NilError(t, err)

		sdk := ebSDKClient(t)

		ruleDeleted := func() (bool, error) {
			resp, err := sdk.ListRulesWithContext(ctx, &eventbridge.ListRulesInput{
				EventBusName: aws.String(busName),
			})
			if err != nil {
				return false, fmt.Errorf("list rules: %w", err)
			}

			return len(resp.Rules) == 0, nil
		}

		waitTimeout := time.Second * 30
		err = wait.For(ruleDeleted, wait.WithTimeout(waitTimeout))
		assert.NilError(t, err, "delete rule: resources not cleaned up in service control plane")

		// delete bus
		var bus ebv1alpha1.EventBus
		err = r.Get(ctx, busName, namespace, &bus)
		assert.NilError(t, err)

		err = r.Delete(ctx, &bus)
		assert.NilError(t, err)

		busDeleted := func() (bool, error) {
			resp, err := sdk.ListEventBusesWithContext(ctx, &eventbridge.ListEventBusesInput{
				NamePrefix: aws.String(busName), // ignore "default" bus
			})
			if err != nil {
				return false, fmt.Errorf("list event buses: %w", err)
			}

			return len(resp.EventBuses) == 0, nil
		}

		err = wait.For(busDeleted, wait.WithTimeout(waitTimeout))
		assert.NilError(t, err, "delete event bus: resources not cleaned up in service control plane")

		return ctx
	}
}
