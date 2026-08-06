package example_test

import (
	"fmt"
	"testing"

	"github.com/mgiaccone/vow"
	"github.com/mgiaccone/vow/example"
)

func ExampleNewCreateServiceArea() {
	cmd, err := example.NewCreateServiceArea("us", "  94103  ")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(cmd.Country, cmd.PostalCode)
	// Output: US 94103
}

func ExampleNewCreateServiceArea_dependentField() {
	_, err := example.NewCreateServiceArea("ca", "94103")
	for _, fe := range vow.FieldErrors(err) {
		fmt.Println(fe)
	}
	// Output: PostalCode: must be a valid Canadian postal code
}

func TestNewCreateServiceArea_Success(t *testing.T) {
	cmd, err := example.NewCreateServiceArea("ca", "K1A 0B1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Country != "CA" || cmd.PostalCode.Unwrap() != "K1A 0B1" {
		t.Fatalf("got %v / %v", cmd.Country, cmd.PostalCode)
	}
}

// TestNewCreateServiceArea_BadCountryReportsOnce is what the c.OK guard buys:
// a bad country must not also produce a fail-closed error against the postal
// code, which would be noise on top of the real complaint.
func TestNewCreateServiceArea_BadCountryReportsOnce(t *testing.T) {
	_, err := example.NewCreateServiceArea("Atlantis", "94103")
	if err == nil {
		t.Fatal("expected an error")
	}
	fes := vow.FieldErrors(err)
	if len(fes) != 1 {
		t.Fatalf("expected exactly 1 field error, got %d: %v", len(fes), fes)
	}
	if fes[0].Field != example.FieldCountry {
		t.Fatalf("expected the error on %s, got %s", example.FieldCountry, fes[0].Field)
	}
}

func TestNewCreateServiceArea_BothFieldsBad(t *testing.T) {
	_, err := example.NewCreateServiceArea("", "")
	fes := vow.FieldErrors(err)
	// The country fails, so the postal code is never attempted: one error,
	// not two.
	if len(fes) != 1 || fes[0].Field != example.FieldCountry {
		t.Fatalf("got %v", fes)
	}
}

func TestNewFindServiceArea(t *testing.T) {
	q, err := example.NewFindServiceArea("K1A 0B1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.PostalCode.Unwrap() != "K1A 0B1" {
		t.Fatalf("got %q", q.PostalCode.Unwrap())
	}

	if _, err := example.NewFindServiceArea("nonsense"); err == nil {
		t.Fatal("expected an error for a malformed postal code")
	}
}
