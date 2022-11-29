package rule

import (
	"fmt"

	"github.com/aws-controllers-k8s/eventbridge-controller/apis/v1alpha1"
)

func validateRuleSpec(spec v1alpha1.RuleSpec) error {
	if s := spec.State; s != nil {
		if !(*s == "ENABLED" || *s == "DISABLED") {
			return fmt.Errorf("invalid Spec: %q must be %q or %q",
				"spec.state",
				"ENABLED",
				"DISABLED",
			)
		}
	}

	emptyPattern := func() bool {
		return spec.EventPattern == nil || *spec.EventPattern == ""
	}

	emptySchedule := func() bool {
		return spec.ScheduleExpression == nil || *spec.ScheduleExpression == ""
	}

	if emptySchedule() && emptyPattern() {
		return fmt.Errorf("invalid Spec: at least one of %q or %q must be specified",
			"spec.eventPattern",
			"spec.scheduleExpression",
		)
	}
	return nil
}
