package fixture

import "github.com/mgiaccone/vow"

type Kind string

const KindDefault Kind = "default"

// The spec takes a discriminator, so parsing needs one and GenerateCode
// carries it. The generator itself still takes nothing.
func codeSpec(k Kind) vow.Spec[string] {
	if k == KindDefault {
		return vow.Spec[string]{Rules: []vow.Rule[string]{vow.NotBlank}}
	}
	return vow.Spec[string]{Rules: []vow.Rule[string]{vow.MinLen(2)}}
}

func codeGenerator() string { return "generated" }

type Code struct {
	v string `vow:"sanitize=trim,json"`
}
