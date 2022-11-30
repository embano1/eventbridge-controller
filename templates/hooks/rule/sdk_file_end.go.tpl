func sdkTargetsFromResource(
	r *resource,
) ([]*svcsdk.Target) {
	var res []*svcsdk.Target
	{{- $field := (index .CRD.SpecFields "Targets" )}}
	for _, krTarget := range r.ko.Spec.Targets {
		t := &svcsdk.Target{}
		{{ GoCodeSetSDKForStruct .CRD "" "t" $field.ShapeRef.Shape.MemberRef "" "krTarget" 1 }}
		res = append(res, t)
	}
	return res
}