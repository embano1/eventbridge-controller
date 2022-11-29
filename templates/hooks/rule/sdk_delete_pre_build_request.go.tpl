
listReq := svcsdk.ListTargetsByRuleInput{
		EventBusName: r.ko.Spec.EventBusName,
		Rule:         r.ko.Spec.Name,
	}
	targets, err := rm.sdkapi.ListTargetsByRuleWithContext(ctx, &listReq)
	if err != nil {
		return nil, err
	}

	var removeTargets []*string
	for _, t := range targets.Targets {
		cp := *t.Id
		removeTargets = append(removeTargets, &cp)
	}

	removeReq := svcsdk.RemoveTargetsInput{
		EventBusName: r.ko.Spec.EventBusName,
		Ids:          removeTargets,
		Rule:         r.ko.Spec.Name,
	}

	// ignoring response as partial failure would lead to reconcile with fresh input from list targets
	_, err = rm.sdkapi.RemoveTargetsWithContext(ctx, &removeReq)
	if err != nil {
		return nil, err
	}
