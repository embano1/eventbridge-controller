func sdkTargetsFromResourceTargets(
	targets []*svcapitypes.Target,
) ([]*svcsdk.Target) {
	var res []*svcsdk.Target
	{{- $field := (index .CRD.SpecFields "Targets" )}}
	for _, krTarget := range targets {
		t := &svcsdk.Target{}
		{{ GoCodeSetSDKForStruct .CRD "" "t" $field.ShapeRef.Shape.MemberRef "" "krTarget" 1 }}
		res = append(res, t)
	}
	return res
}