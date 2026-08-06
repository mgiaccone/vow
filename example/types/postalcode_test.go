package types_test

import (
	"fmt"
	"testing"

	"github.com/mgiaccone/vow/example/types"
)

func ExampleNewPostalCode() {
	p, err := types.NewPostalCode("  k1a 0b1  ", types.CountryCA)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(p)
	// Output: K1A 0B1
}

func ExampleNewAnyPostalCode() {
	p, err := types.NewAnyPostalCode("94103")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(p)
	// Output: 94103
}

// TestPostalCode_DispatchesOnCountry is the point of a spec that takes a
// parameter: the same input is accepted or rejected depending on it, which a
// single Spec var could not express.
func TestPostalCode_DispatchesOnCountry(t *testing.T) {
	cases := []struct {
		name    string
		country types.Country
		in      string
		wantErr string
	}{
		{"us accepts five digits", types.CountryUS, "94103", ""},
		{"us accepts zip+4", types.CountryUS, "94103-1234", ""},
		{"us rejects a canadian code", types.CountryUS, "K1A 0B1", "must be a 5 or 9 digit ZIP code"},
		{"ca accepts a canadian code", types.CountryCA, "K1A 0B1", ""},
		{"ca accepts it unspaced", types.CountryCA, "K1A0B1", ""},
		{"ca rejects a zip", types.CountryCA, "94103", "must be a valid Canadian postal code"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := types.NewPostalCode(c.in, c.country)
			switch {
			case c.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case c.wantErr == "":
				if got.Unwrap() != c.in {
					t.Fatalf("got %q, want %q", got.Unwrap(), c.in)
				}
			case err == nil:
				t.Fatalf("expected error %q, got none", c.wantErr)
			case err.Error() != c.wantErr:
				t.Fatalf("got error %q, want %q", err.Error(), c.wantErr)
			}
		})
	}
}

// TestPostalCode_FailsClosed covers the trap the feature introduces: a spec
// func's default arm must reject. Returning a zero vow.Spec there would have
// no rules and quietly accept anything.
func TestPostalCode_FailsClosed(t *testing.T) {
	// A conversion to a defined type is always legal in Go, so an unknown
	// country is reachable and must not open the gate.
	_, err := types.NewPostalCode("literally anything", types.Country("Atlantis"))
	if err == nil {
		t.Fatal("an unknown country must fail closed")
	}
	if err.Error() != "has an unknown country" {
		t.Fatalf("got %q", err.Error())
	}
}

func TestPostalCode_SanitizesBeforeValidating(t *testing.T) {
	p, err := types.NewPostalCode("  k1a 0b1  ", types.CountryCA)
	if err != nil {
		t.Fatal(err)
	}
	if p.Unwrap() != "K1A 0B1" {
		t.Fatalf("sanitize=trim|upper not applied: %q", p.Unwrap())
	}
}

func TestMustPostalCode_PanicsOnInvalid(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic")
		}
	}()
	types.MustPostalCode("nonsense", types.CountryUS)
}

// TestAnyPostalCode_UnionPath is the read side: a coverage check carries a
// postal code with no country, so it can only be checked as well-formed for
// some supported country.
func TestAnyPostalCode_UnionPath(t *testing.T) {
	for _, in := range []string{"94103", "94103-1234", "K1A 0B1"} {
		if _, err := types.NewAnyPostalCode(in); err != nil {
			t.Errorf("NewAnyPostalCode(%q) unexpected error: %v", in, err)
		}
	}
	if _, err := types.NewAnyPostalCode("nonsense"); err == nil {
		t.Fatal("expected an error for a malformed postal code")
	}
}

// TestAnyPostalCodeAndPostalCodeAreDistinct records the guarantee that the
// separate types exist for. It is a compile-time property, so the test can
// only state it: AnyPostalCode is not assignable to PostalCode, which is what
// stops a lookup key reaching a write path. Uncommenting the line below is a
// compile error.
func TestAnyPostalCodeAndPostalCodeAreDistinct(t *testing.T) {
	lookup := types.MustAnyPostalCode("94103")
	stored := types.MustPostalCode("94103", types.CountryUS)

	// var _ types.PostalCode = lookup   // compile error, by design

	if lookup.Unwrap() != stored.Unwrap() {
		t.Fatal("both wrap the same string; only their guarantees differ")
	}
}
