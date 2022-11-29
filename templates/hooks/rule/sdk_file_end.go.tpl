func (rm *resourceManager) KRtoSDK(
	r *resource,
) ([]*svcsdk.Target, error) {
	var res []*svcsdk.Target
	for _, krTarget := range r.ko.Spec.Targets {
		t := &svcsdk.Target{}
		{{ GoCodeSetSDKForStruct .CRD "" "t" $field.ShapeRef.Shape.MemberRef "" "krTarget" 1 }}
		res = append(res, t)
	}
	return res, nil
}
