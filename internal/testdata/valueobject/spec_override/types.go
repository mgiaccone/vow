package fixture

import "github.com/mgiaccone/vow"

var sharedSpec = vow.Spec[string]{
	Rules: []vow.Rule[string]{vow.NotBlank},
}

type Handle struct {
	v string `vow:"spec=sharedSpec,json"`
}
