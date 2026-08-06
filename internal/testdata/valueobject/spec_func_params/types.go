package fixture

import "github.com/mgiaccone/vow"

type Kind string

// codeSpec exercises the named parameter shapes: grouped names sharing a
// type, a blank name, and a trailing variadic. Grouped names expand to one
// parameter each; a blank one is given a positional name, since a generated
// signature needs something to call it; the variadic is forwarded with "..."
// so the caller's argument list survives.
func codeSpec(a, b Kind, _ int, extra ...Kind) vow.Spec[string] {
	return vow.Spec[string]{Rules: []vow.Rule[string]{vow.NotBlank}}
}

type Code struct {
	v string `vow:"json"`
}
