package e2e

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/kelseyhightower/envconfig"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/pkg/features"

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

type namespaceCtxKey string

const (
	baseCRDPath   = "../../config/crd/bases"
	commonCRDPath = "../../config/crd/common/bases"
	namespaceKey  = namespaceCtxKey("featureNamespace")
)

var (
	testEnv env.Environment
	envCfg  envConfig

	// test queue
	queueName string
	queueARN  string
	queueURL  string

	// common tags
	tags []*ebv1alpha1.Tag
)

func TestMain(m *testing.M) {
	envconfig.MustProcess("", &envCfg)

	cfg, err := envconf.NewFromFlags()
	if err != nil {
		log.Fatalf("envconf failed: %s", err)
	}

	testEnv = env.NewWithConfig(cfg)
	queueName = envconf.RandomName("ack-e2e-queue", 20)

	tags = []*ebv1alpha1.Tag{{
		Key:   aws.String("ack-e2e"),
		Value: aws.String("true"),
	}}

	klog.V(1).Infof("setting up test environment with kind cluster %q", envCfg.KindCluster)
	testEnv.Setup(
		createSQSTestQueue(queueName, tags),
		envfuncs.CreateKindCluster(envCfg.KindCluster),
		envfuncs.SetupCRDs(baseCRDPath, "*"),
		envfuncs.SetupCRDs(commonCRDPath, "*"),
	)

	testEnv.Finish(
		envfuncs.DeleteNamespace(ackNamespace),
		destroySQSTestQueue(),
	)

	// create/delete namespace per feature
	testEnv.BeforeEachFeature(func(ctx context.Context, cfg *envconf.Config, _ *testing.T, f features.Feature) (context.Context, error) {
		return createNSForFeature(ctx, cfg, f.Name())
	})
	testEnv.AfterEachFeature(func(ctx context.Context, cfg *envconf.Config, t *testing.T, f features.Feature) (context.Context, error) {
		return deleteNSForFeature(ctx, cfg, t, f.Name())
	})

	os.Exit(testEnv.Run(m))
}

// createNSForFeature creates a random namespace with the runID as a prefix. It is stored in the context
// so that the deleteNSForFeature routine can look it up and delete it.
func createNSForFeature(ctx context.Context, cfg *envconf.Config, feature string) (context.Context, error) {
	ns := envconf.RandomName("ack-feature", 15)
	ctx = context.WithValue(ctx, namespaceKey, ns)

	klog.V(1).Infof("creating namespace %q for feature %q", ns, feature)
	nsObj := corev1.Namespace{}
	nsObj.Name = ns

	return ctx, cfg.Client().Resources().Create(ctx, &nsObj)
}

// deleteNSForFeature looks up the namespace corresponding to the given test and deletes it.
func deleteNSForFeature(ctx context.Context, cfg *envconf.Config, t *testing.T, feature string) (context.Context, error) {
	ns := getTestNamespaceFromContext(ctx, t)

	klog.V(1).Infof("deleting namespace %q for feature %q", ns, feature)

	nsObj := corev1.Namespace{}
	nsObj.Name = ns

	return ctx, cfg.Client().Resources().Delete(ctx, &nsObj)
}

func getTestNamespaceFromContext(ctx context.Context, t *testing.T) string {
	ns, ok := ctx.Value(namespaceKey).(string)
	assert.Equal(t, ok, true, "retrieve namespace from context: value not found for key %q", namespaceKey)
	return ns
}
