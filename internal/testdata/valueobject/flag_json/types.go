package fixture

import "github.com/mgiaccone/vow"

var labelSpec = vow.Spec[string]{
	Rules: []vow.Rule[string]{vow.NotBlank},
}

type Label struct {
	v string `vow:"json"`
}
