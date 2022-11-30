	if len(ko.Spec.Targets) > 0 {
		err := rm.syncRuleTargets(
			ctx,
			ko.Spec.Name, ko.Spec.EventBusName,
			ko.Spec.Targets, nil,
		)
		if err != nil {
			return nil, err
		}
	}