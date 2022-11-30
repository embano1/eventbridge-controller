	if err = validateRuleSpec(desired.ko.Spec); err != nil {
			return nil, ackerr.NewTerminalError(err)
	}
	if delta.DifferentAt("Spec.Tags") {
		err = rm.syncRuleTags(ctx, desired, latest)
		if err != nil {
			return nil, err
		}
	}
	if delta.DifferentAt("Spec.Targets") {
		err = rm.syncRuleTargets(
			ctx,
			desired.ko.Spec.Name, desired.ko.Spec.EventBusName,
			desired.ko.Spec.Targets, latest.ko.Spec.Targets,
		)
		if err != nil {
			return nil, err
		}
	}
	if !delta.DifferentExcept("Spec.Tags", "Spec.Targets") {
		return desired, nil
	}