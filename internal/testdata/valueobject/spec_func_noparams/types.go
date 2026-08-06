package fixture

import "github.com/mgiaccone/vow"

// A spec func taking no parameters is the degenerate case of the func form,
// and it must be called: emailSpec is a func value, so emailSpec.Sanitizing
// would not compile. Taking no parameters, it is still hoistable, so this
// generates a full parser exactly as a var spec does.
func emailSpec() vow.Spec[string] {
	return vow.Spec[string]{Rules: []vow.Rule[string]{vow.NotBlank, vow.MaxLen(254)}}
}

type Email struct {
	v string `vow:"sanitize=trim|lower,json"`
}

// Without sanitize= there is no parser to hoist, so the call appears inline
// as codeSpec().Parse(in) — the other half of the same requirement.
func codeSpec() vow.Spec[string] {
	return vow.Spec[string]{Rules: []vow.Rule[string]{vow.NotBlank}}
}

type Code struct {
	v string `vow:"json"`
}
