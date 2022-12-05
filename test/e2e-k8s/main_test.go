package e2e

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/kelseyhightower/envconfig"
	"k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"

	ebv1alpha1 "github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
)

type envConfig struct {
	Region       string `envconfig:"AWS_DEFAULT_REGION" required:"true"`
	AccessKey    string `envconfig:"AWS_ACCESS_KEY_ID" required:"true"`
	SecretKey    string `envconfig:"AWS_SECRET_ACCESS_KEY" required:"true"`
	SessionToken string `envconfig:"AWS_SESSION_TOKEN" required:"true"`
	KindCluster  string `envconfig:"KIND_CLUSTER_NAME" default:"ack"`
	CtrlImage    string `envconfig:"ACK_CONTROLLER_IMAGE" required:"true"`
}

const (
	baseCRDPath   = "../../config/crd/bases"
	commonCRDPath = "../../config/crd/common/bases"
)

var (
	testEnv env.Environment
	envCfg  envConfig

	// test queue
	queueName string
	queueARN  string
	queueURL  string

	testNamespace string
	testBusName   string
	testRuleName  string
	tags          []*ebv1alpha1.Tag
)

func TestMain(m *testing.M) {
	envconfig.MustProcess("", &envCfg)

	cfg, err := envconf.NewFromFlags()
	if err != nil {
		log.Fatalf("envconf failed: %s", err)
	}

	testEnv = env.NewWithConfig(cfg)

	testNamespace = envconf.RandomName("ack-e2e", 20)
	queueName = envconf.RandomName("ack-e2e-queue", 20)
	testBusName = envconf.RandomName("ack-bus-e2e", 20)
	testRuleName = envconf.RandomName("ack-rule-e2e", 20)

	tags = []*ebv1alpha1.Tag{{
		Key:   aws.String("ack-e2e"),
		Value: aws.String("true"),
	}}

	klog.V(1).Infof("setting up test environment with kind cluster %q", envCfg.KindCluster)
	testEnv.Setup(
		createSQSTestQueue(queueName, tags),
		envfuncs.CreateKindCluster(envCfg.KindCluster),
		envfuncs.CreateNamespace(testNamespace),
		envfuncs.SetupCRDs(baseCRDPath, "*"),
		envfuncs.SetupCRDs(commonCRDPath, "*"),
	)

	testEnv.Finish(
		/*envfuncs.DeleteNamespace(ackNamespace),
		envfuncs.TeardownCRDs(baseCRDPath, "*"),
		envfuncs.DestroyKindCluster(kindClusterName),*/
		destroySQSTestQueue(),
	)

	os.Exit(testEnv.Run(m))
}

func createSQSTestQueue(name string, tags []*ebv1alpha1.Tag) env.Func {
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

		resp, err := sqssdk.CreateQueue(&sqs.CreateQueueInput{
			QueueName: aws.String(name),
			Tags:      sqstags,
		})
		if err != nil {
			return ctx, fmt.Errorf("create sqs queue: %w", err)
		}
		queueURL = *resp.QueueUrl
		klog.V(1).Infof("created test sqs queue %q", *resp.QueueUrl)

		const arnKey = "QueueArn"
		attrs, err := sqssdk.GetQueueAttributes(&sqs.GetQueueAttributesInput{
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

		_, err = sqssdk.DeleteQueue(&sqs.DeleteQueueInput{
			QueueUrl: aws.String(queueURL),
		})
		if err != nil {
			return ctx, fmt.Errorf("destroy sqs queue: %w", err)
		}

		klog.V(1).Infof("destroyed test sqs queue %q", queueURL)
		return ctx, nil
	}
}
