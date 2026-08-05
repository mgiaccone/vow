package main

import (
	"strings"
	"unicode"
)

// specVarName derives the conventional spec variable name for a value
// object type: lowercase the leading run of capitals, keeping the capital
// that starts the next word, and append "Spec".
//
//	Email   -> emailSpec
//	URLPath -> urlPathSpec
//	URL     -> urlSpec
func specVarName(typeName string) string {
	return lowerLeadingRun(typeName) + "Spec"
}

func lowerLeadingRun(name string) string {
	runes := []rune(name)
	n := len(runes)
	i := 0
	for i < n && unicode.IsUpper(runes[i]) {
		i++
	}
	switch {
	case i == 0:
		// Doesn't start with an uppercase letter; nothing to do. Exported
		// Go identifiers always do, so this is defensive only.
		return name
	case i == n:
		// Entirely uppercase, e.g. "URL": lowercase the whole thing.
		return strings.ToLower(name)
	case i == 1:
		// Only the first letter is uppercase, e.g. "Email".
		return strings.ToLower(string(runes[0])) + string(runes[1:])
	default:
		// A run of several capitals followed by a lowercase letter, e.g.
		// "URLPath": keep the last capital as the start of the next word.
		return strings.ToLower(string(runes[:i-1])) + string(runes[i-1:])
	}
}
