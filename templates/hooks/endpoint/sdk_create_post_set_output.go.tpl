	if !endpointAvailable(&resource{ko}) {
		// requeue: endpoint usually not immediately available i.e., CREATING or CREATE_FAILED
		return &resource{ko}, requeueWaitWhileCreating
	}
