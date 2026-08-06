package fixture

import "github.com/mgiaccone/vow"

type Kind string

// Go requires parameter names to be all present or all absent, so unnamed
// parameters get their own case. The generator names them positionally,
// because a generated signature cannot leave them anonymous and still call
// through to the spec.
func codeSpec(Kind, int) vow.Spec[string] {
	return vow.Spec[string]{Rules: []vow.Rule[string]{vow.NotBlank}}
}

type Code struct {
	v string `vow:"json"`
}
