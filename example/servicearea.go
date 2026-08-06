package example

import (
	"github.com/mgiaccone/vow"
	"github.com/mgiaccone/vow/example/types"
)

const (
	FieldCountry    vow.Field = "Country"
	FieldPostalCode vow.Field = "PostalCode"
)

// CreateServiceArea is the write path: the payload carries both the country
// and the postal code, so the postal code can be validated against it.
type CreateServiceArea struct {
	Country    types.Country
	PostalCode types.PostalCode
}

// NewCreateServiceArea shows a field whose validity depends on another field.
// The country is parsed first; the postal code is parsed with it, and only
// once the country is known good.
func NewCreateServiceArea(rawCountry, rawPostalCode string) (CreateServiceArea, error) {
	var c vow.Collector

	cmd := CreateServiceArea{
		Country: vow.Collect(&c, FieldCountry, rawCountry, types.NewCountry),
	}

	// The postal code cannot go in the literal above: it needs the country.
	// Guarding on c.OK keeps a bad country from producing a second,
	// misleading error here from postalCodeSpec's fail-closed default — the
	// real complaint is about the country, and it is already recorded.
	if c.OK(FieldCountry) {
		cmd.PostalCode = vow.CollectFunc(&c, FieldPostalCode, func() (types.PostalCode, error) {
			return types.NewPostalCode(rawPostalCode, cmd.Country)
		})
	}

	return cmd, c.Err()
}

// FindServiceArea is the read path: a coverage check carries a postal code
// but no country, so the postal code can only be checked as well-formed for
// some supported country. Its field is an AnyPostalCode, which cannot be
// passed where a PostalCode is required.
type FindServiceArea struct {
	PostalCode types.AnyPostalCode
}

func NewFindServiceArea(rawPostalCode string) (FindServiceArea, error) {
	var c vow.Collector
	q := FindServiceArea{
		PostalCode: vow.Collect(&c, FieldPostalCode, rawPostalCode, types.NewAnyPostalCode),
	}
	return q, c.Err()
}
