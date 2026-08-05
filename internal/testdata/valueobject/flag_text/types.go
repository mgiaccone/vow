package fixture

import "github.com/mgiaccone/vow"

var tokenSpec = vow.Spec[string]{
	Rules: []vow.Rule[string]{vow.NotBlank},
}

type Token struct {
	v string `vow:"text"`
}
