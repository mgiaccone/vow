package fixture

import "github.com/mgiaccone/vow"

type Kind string

const KindDefault Kind = "default"

// With sanitize= on the tag, a func spec cannot hoist a whole parser the way
// a var spec does — the Spec depends on k — so only the sanitizer chain is
// hoisted, into codeSanitizer.
func codeSpec(k Kind) vow.Spec[string] {
	if k == KindDefault {
		return vow.Spec[string]{Rules: []vow.Rule[string]{vow.NotBlank}}
	}
	return vow.Spec[string]{Rules: []vow.Rule[string]{vow.MinLen(2)}}
}

type Code struct {
	v string `vow:"sanitize=trim|lower,json,sql,text"`
}
