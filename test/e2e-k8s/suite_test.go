//go:build e2e

package e2e

import (
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
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
	testBusName     string
	testRuleName    string
	testArchiveName string
)

func TestSuite(t *testing.T) {
	testBusName = envconf.RandomName("ack-bus-e2e", 20)
	testRuleName = envconf.RandomName("ack-rule-e2e", 20)
	testArchiveName = envconf.RandomName("ack-archive-e2e", 20)

	// required for other features
	ctrl := features.New("EventBridge Controller").
		Setup(createController()).
		Assess("controller running without leader election", controllerRunning()).
		Feature()

	/*bus := features.New("EventBridge Event Bus CRUD").
		Assess("create event bus", createEventBus(testBusName, tags)).
		Assess("event bus has synced", eventBusSynced(testBusName, tags)).
		Assess("update event bus", updateEventBus(testBusName)).
		Assess("delete event bus", deleteBus(testBusName)).
		Feature()

	rule := features.New("EventBridge Rule CRUD").
		Setup(setupBus(testBusName, tags)).
		Assess("create rule", createRule(testRuleName, testBusName, tags)).
		Assess("rule has synced", ruleSynced(testRuleName, testBusName, tags)).
		Assess("update rule", updateRule(testRuleName, testBusName)).
		Assess("delete rule", deleteRule(testRuleName, testBusName)).
		Teardown(deleteBus(testBusName)).
		Feature()

	invalidRule := features.New("EventBridge Rule invalid in terminal state").
		Setup(setupBus(testBusName, tags)).
		Assess("create invalid rule", createInvalidRule(testRuleName, testBusName, tags)).
		Assess("rule is in terminal state", ruleInTerminalState(testRuleName, testBusName, tags)).
		Assess("delete rule", deleteRule(testRuleName, testBusName)).
		Teardown(deleteBus(testBusName)).
		Feature()*/

	archive := features.New("EventBridge Archive CRUD").
		Setup(setupBus(testBusName, tags)).
		Assess("create archive", createArchive(testArchiveName)).
		Assess("archive has synced", archiveSynced(testArchiveName)).
		Assess("update archive", updateArchive(testArchiveName)).
		Assess("delete archive", deleteArchive(testArchiveName)).
		Teardown(deleteBus(testBusName)).
		Feature()

	/*	e2e := features.New("EventBridge E2E").
		Setup(setupBus(testBusName, tags)).
		Assess("create rule", createRule(testRuleName, testBusName, tags)).
		Assess("rule has synced", ruleSynced(testRuleName, testBusName, tags)).
		Assess("event received in sqs", eventReceived(testBusName)).
		Assess("delete rule", deleteRule(testRuleName, testBusName)).
		Teardown(deleteBus(testBusName)).
		Feature()
	*/
	testEnv.Test(t, ctrl, archive)
}
