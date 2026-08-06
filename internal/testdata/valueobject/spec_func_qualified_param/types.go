package fixture

import (
	"time"

	"github.com/mgiaccone/vow"
)

// A qualified parameter type must pull its package into the generated
// file's imports, the same way a qualified base type does. The base here is
// a plain string, so "time" can only reach the output through the parameter.
func codeSpec(at time.Time) vow.Spec[string] {
	return vow.Spec[string]{Rules: []vow.Rule[string]{vow.NotBlank}}
}

type Code struct {
	v string `vow:"json"`
}
