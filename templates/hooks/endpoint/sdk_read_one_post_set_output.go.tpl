
	/*res := &resource{ko}
	if endpointInTerminalState(res) {
		msg := "Endpoint is in '" + *res.ko.Status.State + "' status"
		ackcondition.SetTerminal(res, corev1.ConditionTrue, &msg, res.ko.Status.StateReason)
		ackcondition.SetSynced(res, corev1.ConditionTrue, nil, nil)
		return res, nil
	}*/
