	if len(ko.Spec.Targets) > 0 {
		if err := rm.syncRuleTargets(ko.Spec.Targets, nil); err != nil {
			return &resource{ko}, err
		}
	}