
	// eventually consistent response, only use status and always requeue
	res, err := rm.ReadOne(ctx, desired)
	if err != nil {
		return nil, err
	}

	desired.SetStatus(res)
	return desired, ackrequeue.NeededAfter(nil, defaultRequeueDelay)
