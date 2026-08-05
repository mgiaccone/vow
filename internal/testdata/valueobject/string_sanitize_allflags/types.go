package fixture

import (
	"regexp"

	"github.com/mgiaccone/vow"
)

const maxEmailLen = 254

var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

var emailSpec = vow.Spec[string]{
	Rules: []vow.Rule[string]{
		vow.NotBlank,
		vow.MaxLen(maxEmailLen),
		vow.Matches(emailPattern, "must be a valid email address"),
	},
}

type Email struct {
	v string `vow:"sanitize=trim|lower,json,sql,text"`
}
