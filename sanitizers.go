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

// Collapse replaces every run of whitespace, including leading and trailing
// runs, with a single space.
func Collapse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
