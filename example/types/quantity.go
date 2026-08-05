package types

import "github.com/mgiaccone/vow"

var quantitySpec = vow.Spec[int]{
	Rules: []vow.Rule[int]{vow.Positive[int]},
}

// Quantity is a value object for a strictly positive count.
type Quantity struct {
	v int `vow:"json,sql"`
}
