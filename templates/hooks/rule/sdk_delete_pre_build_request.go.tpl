	if _, err := rm.preDeleteRule(ctx, &resource{ko}); err != nil {
		return nil, err
	}