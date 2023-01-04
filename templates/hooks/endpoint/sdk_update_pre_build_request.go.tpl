
    if err = validateEndpointSpec(delta,desired.ko.Spec); err != nil {
    		return nil, ackerr.NewTerminalError(err)
    }

    if endpointInTerminalState(latest) {
		msg := "Endpoint is in '" + *latest.ko.Status.State + "' status"
		ackcondition.SetTerminal(desired, corev1.ConditionTrue, &msg, nil)
		ackcondition.SetSynced(desired, corev1.ConditionTrue, nil, nil)
		return desired, nil
	}

	if endpointCreating(latest) {
		msg := "Endpoint is currently being created"
		ackcondition.SetSynced(desired, corev1.ConditionFalse, &msg, nil)
		return desired, requeueWaitUntilCanModify(latest)
	}

	if !endpointAvailable(latest) {
		msg := "Endpoint is not available for modification in '" +
			*latest.ko.Status.State + "' status"
		ackcondition.SetSynced(desired, corev1.ConditionFalse, &msg, nil)
		return desired, requeueWaitUntilCanModify(latest)
	}
