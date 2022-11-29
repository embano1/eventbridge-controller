	if len(ko.Spec.Targets) > 0 {
		if err := rm.addRuleTargets(ctx, ko); err != nil {
			return &resource{ko}, err
		}
	}