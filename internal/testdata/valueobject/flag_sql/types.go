package fixture

import "github.com/mgiaccone/vow"

var codeSpec = vow.Spec[string]{
	Rules: []vow.Rule[string]{vow.NotBlank},
}

type Code struct {
	v string `vow:"sql"`
}
