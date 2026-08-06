package types

import (
	"errors"
	"regexp"

	"github.com/mgiaccone/vow"
)

// PhoneNumberType discriminates how a number must be shaped. It is an input
// to validation, never part of a number's value — which is why it is a
// parameter of typedNumberSpec below rather than a field on TypedNumber.
//
//vow:enum sanitize=trim|lower, json, sql
type PhoneNumberType string

const (
	TypeShortCode PhoneNumberType = "shortcode"
	TypeLVN       PhoneNumberType = "lvn"
)

var (
	shortCodePattern = regexp.MustCompile(`^[0-9]{4,6}$`)
	lvnPattern       = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)
	anyNumberPattern = regexp.MustCompile(`^(\+[1-9][0-9]{7,14}|[0-9]{4,6})$`)

	shortCodeSpec = vow.Spec[string]{Rules: []vow.Rule[string]{
		vow.NotBlank,
		vow.Matches(shortCodePattern, "must be 4 to 6 digits"),
	}}

	lvnSpec = vow.Spec[string]{Rules: []vow.Rule[string]{
		vow.NotBlank,
		vow.Matches(lvnPattern, "must be a valid E.164 number"),
	}}

	errUnknownType = errors.New("has an unknown phone number type")

	unknownTypeSpec = vow.Spec[string]{Rules: []vow.Rule[string]{
		func(string) error { return errUnknownType },
	}}
)

// anyNumberSpec is a plain var, so AnyNumber generates the ordinary
// single-argument constructor.
var anyNumberSpec = vow.Spec[string]{Rules: []vow.Rule[string]{
	vow.NotBlank,
	vow.Matches(anyNumberPattern, "must be a valid number"),
}}

// AnyNumber is validated only as "well-formed as some kind of number". It is
// a lookup key — a GET request carries a number but no type — and must never
// be persisted, which is why it is a distinct type from TypedNumber rather
// than a second constructor on one type. The compiler then refuses to let a
// lookup key reach a write path.
type AnyNumber struct {
	v string `vow:"sanitize=trim,json,sql"`
}

// typedNumberSpec is a func rather than a var. That is the whole mechanism:
// its parameters are appended to the generated constructors, so TypedNumber
// cannot be built without saying which kind it must satisfy.
//
// The default arm fails closed on purpose. Returning a zero vow.Spec would
// have no rules and therefore accept everything.
func typedNumberSpec(t PhoneNumberType) vow.Spec[string] {
	switch t {
	case TypeShortCode:
		return shortCodeSpec
	case TypeLVN:
		return lvnSpec
	default:
		return unknownTypeSpec
	}
}

// TypedNumber is validated against a known kind. This is what gets stored.
type TypedNumber struct {
	v string `vow:"sanitize=trim,json,sql,text"`
}
