package e2e

import (
	"log"
	"os"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
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
	testEnv     env.Environment
	crNamespace string
	envCfg      envConfig
)

func TestMain(m *testing.M) {
	envconfig.MustProcess("", &envCfg)

	cfg := envconf.NewWithKubeConfig("/Users/mgasch/.kube/config")
	testEnv = env.NewWithConfig(cfg)

	crNamespace = envconf.RandomName("ack-test", 20)

	log.Printf("setting up test environment with kind cluster %q", envCfg.KindCluster)
	testEnv.Setup(
		envfuncs.CreateKindCluster(envCfg.KindCluster),
		envfuncs.CreateNamespace(crNamespace),
		envfuncs.SetupCRDs(baseCRDPath, "*"),
		envfuncs.SetupCRDs(commonCRDPath, "*"),
	)

	/*testEnv.Finish(
		envfuncs.DeleteNamespace(ackNamespace),
		envfuncs.TeardownCRDs(baseCRDPath, "*"),
		envfuncs.DestroyKindCluster(kindClusterName),
	)*/

	os.Exit(testEnv.Run(m))
}
