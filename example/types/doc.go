// Package types declares the value objects and enum used by the example
// command in the parent package. It exists to exercise vow's generator
// end to end, not to model a real domain — see the CreateInvite command in
// the parent package for the interesting part: a cross-field rule that a
// per-type generator can't express on its own.
//
//go:generate go tool vow -dir=.
package types
