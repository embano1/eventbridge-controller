	if len(ko.Spec.Tags) > 0 {
		return &resource{ko}, requeueOnCreate
	}