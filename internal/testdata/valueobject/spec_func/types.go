package fixture

import "github.com/mgiaccone/vow"

// Kind is the discriminator: an input to validation, never part of Code.
type Kind string

const (
	KindShort Kind = "short"
	KindLong  Kind = "long"
)

// codeSpec is a func rather than a var, which is what gives NewCode its
// extra parameter.
func codeSpec(k Kind) vow.Spec[string] {
	switch k {
	case KindShort:
		return vow.Spec[string]{Rules: []vow.Rule[string]{vow.MaxLen(4)}}
	case KindLong:
		return vow.Spec[string]{Rules: []vow.Rule[string]{vow.MinLen(8)}}
	default:
		return vow.Spec[string]{Rules: []vow.Rule[string]{vow.NotBlank}}
	}
}

type Code struct {
	v string `vow:"json"`
}
