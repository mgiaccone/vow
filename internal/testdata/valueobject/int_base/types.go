package fixture

import "github.com/mgiaccone/vow"

var quantitySpec = vow.Spec[int]{
	Rules: []vow.Rule[int]{vow.Positive[int]},
}

type Quantity struct {
	v int `vow:"sql"`
}
