package types

import (
	"errors"
	"regexp"

	"github.com/mgiaccone/vow"
)

// Country discriminates how a postal code must be shaped. It is an input to
// validation, never part of a postal code's value — which is why it is a
// parameter of postalCodeSpec below rather than a field on PostalCode.
//
//vow:enum sanitize=trim|upper, json, sql
type Country string

const (
	CountryUS Country = "US"
	CountryCA Country = "CA"
)

var (
	usPostalCodePattern = regexp.MustCompile(`^[0-9]{5}(-[0-9]{4})?$`)

	// The space is optional here on purpose: sanitize=trim|upper normalizes
	// case, not spacing, so "K1A0B1" has to be accepted as written.
	caPostalCodePattern = regexp.MustCompile(`^[A-Z][0-9][A-Z] ?[0-9][A-Z][0-9]$`)

	anyPostalCodePattern = regexp.MustCompile(`^([0-9]{5}(-[0-9]{4})?|[A-Z][0-9][A-Z] ?[0-9][A-Z][0-9])$`)

	usPostalCodeSpec = vow.Spec[string]{Rules: []vow.Rule[string]{
		vow.NotBlank,
		vow.Matches(usPostalCodePattern, "must be a 5 or 9 digit ZIP code"),
	}}

	caPostalCodeSpec = vow.Spec[string]{Rules: []vow.Rule[string]{
		vow.NotBlank,
		vow.Matches(caPostalCodePattern, "must be a valid Canadian postal code"),
	}}

	errUnknownCountry = errors.New("has an unknown country")

	unknownCountrySpec = vow.Spec[string]{Rules: []vow.Rule[string]{
		func(string) error { return errUnknownCountry },
	}}
)

// anyPostalCodeSpec takes no parameters, so it is written as a plain var and
// AnyPostalCode generates the ordinary single-argument constructor.
var anyPostalCodeSpec = vow.Spec[string]{Rules: []vow.Rule[string]{
	vow.NotBlank,
	vow.Matches(anyPostalCodePattern, "must be a valid postal code"),
}}

// AnyPostalCode is validated only as "well-formed for some supported
// country". It is a lookup key — a coverage check carries a postal code but
// no country — and must never be persisted, which is why it is a distinct
// type from PostalCode rather than a second constructor on one type. The
// compiler then refuses to let a lookup key reach a write path.
type AnyPostalCode struct {
	v string `vow:"sanitize=trim|upper,json,sql"`
}

// postalCodeSpec takes a parameter, which is the whole mechanism: the
// parameter is appended to the generated constructors, so a PostalCode
// cannot be built without saying which country it must satisfy.
//
// The default arm fails closed on purpose. Returning a zero vow.Spec would
// have no rules and therefore accept everything.
func postalCodeSpec(c Country) vow.Spec[string] {
	switch c {
	case CountryUS:
		return usPostalCodeSpec
	case CountryCA:
		return caPostalCodeSpec
	default:
		return unknownCountrySpec
	}
}

// PostalCode is validated against a known country. This is what gets stored,
// and it takes the unqualified name so that reaching for the obvious type
// gets the stronger guarantee.
type PostalCode struct {
	v string `vow:"sanitize=trim|upper,json,sql,text"`
}
