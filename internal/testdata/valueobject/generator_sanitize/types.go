package fixture

import "github.com/mgiaccone/vow"

var codeSpec = vow.Spec[string]{Rules: []vow.Rule[string]{vow.NotBlank}}

func codeGenerator() string { return "  GENERATED  " }

// With sanitize= the generated value goes through the hoisted parser, so a
// generator is held to the same normalization as parsed input.
type Code struct {
	v string `vow:"sanitize=trim|lower,json,sql"`
}
