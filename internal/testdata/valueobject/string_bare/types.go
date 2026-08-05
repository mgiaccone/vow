package fixture

import "github.com/mgiaccone/vow"

var emailSpec = vow.Spec[string]{
	Rules: []vow.Rule[string]{vow.NotBlank},
}

type Email struct {
	v string `vow:""`
}
