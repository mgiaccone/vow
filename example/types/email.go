package types

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

// Email is a value object for a syntactically valid email address. It is
// normalized to lowercase with surrounding whitespace trimmed before
// validation, so "  User@Example.com " and "user@example.com" produce the
// same Email.
type Email struct {
	v string `vow:"sanitize=trim|lower,json,sql,text"`
}
