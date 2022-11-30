	if len(r.ko.Spec.Targets) > 0 {
		err := rm.syncRuleTargets(
			ctx,
			r.ko.Spec.Name, r.ko.Spec.EventBusName,
			nil, r.ko.Spec.Targets,
		)
		if err != nil {
			return nil, err
		}
	}