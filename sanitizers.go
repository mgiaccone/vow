package vow

import "strings"

// Trim removes leading and trailing whitespace.
func Trim(s string) string {
	return strings.TrimSpace(s)
}

// Lower lowercases s.
func Lower(s string) string {
	return strings.ToLower(s)
}

// Upper uppercases s.
func Upper(s string) string {
	return strings.ToUpper(s)
}

// Collapse reduces every interior run of whitespace to a single space and
// removes leading and trailing whitespace entirely, so it subsumes Trim.
func Collapse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
