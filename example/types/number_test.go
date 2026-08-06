package types_test

import (
	"fmt"
	"testing"

	"github.com/mgiaccone/vow/example/types"
)

func ExampleNewTypedNumber() {
	n, err := types.NewTypedNumber("  12345  ", types.TypeShortCode)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(n)
	// Output: 12345
}

func ExampleNewAnyNumber() {
	n, err := types.NewAnyNumber("+14155550123")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(n)
	// Output: +14155550123
}

// TestTypedNumber_DispatchesOnKind is the point of a func spec: the same
// input is accepted or rejected depending on the parameter, which a single
// Spec var could not express.
func TestTypedNumber_DispatchesOnKind(t *testing.T) {
	cases := []struct {
		name    string
		kind    types.PhoneNumberType
		in      string
		wantErr string
	}{
		{"shortcode accepts digits", types.TypeShortCode, "12345", ""},
		{"shortcode rejects E.164", types.TypeShortCode, "+14155550123", "must be 4 to 6 digits"},
		{"lvn accepts E.164", types.TypeLVN, "+14155550123", ""},
		{"lvn rejects a shortcode", types.TypeLVN, "12345", "must be a valid E.164 number"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := types.NewTypedNumber(c.in, c.kind)
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

// TestTypedNumber_FailsClosed covers the trap the feature introduces: a spec
// func's default arm must reject. Returning a zero vow.Spec there would have
// no rules and quietly accept anything.
func TestTypedNumber_FailsClosed(t *testing.T) {
	// A conversion to a defined type is always legal in Go, so an unknown
	// kind is reachable and must not open the gate.
	_, err := types.NewTypedNumber("literally anything", types.PhoneNumberType("carrier-pigeon"))
	if err == nil {
		t.Fatal("an unknown kind must fail closed")
	}
	if err.Error() != "has an unknown phone number type" {
		t.Fatalf("got %q", err.Error())
	}
}

func TestTypedNumber_SanitizesBeforeValidating(t *testing.T) {
	n, err := types.NewTypedNumber("  12345  ", types.TypeShortCode)
	if err != nil {
		t.Fatal(err)
	}
	if n.Unwrap() != "12345" {
		t.Fatalf("sanitize=trim not applied: %q", n.Unwrap())
	}
}

func TestMustTypedNumber_PanicsOnInvalid(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic")
		}
	}()
	types.MustTypedNumber("nonsense", types.TypeShortCode)
}

// TestAnyNumber_UnionPath is the read side: a GET carries a number with no
// kind, so it can only be checked as well-formed for some kind.
func TestAnyNumber_UnionPath(t *testing.T) {
	for _, in := range []string{"12345", "+14155550123"} {
		if _, err := types.NewAnyNumber(in); err != nil {
			t.Errorf("NewAnyNumber(%q) unexpected error: %v", in, err)
		}
	}
	if _, err := types.NewAnyNumber("nonsense"); err == nil {
		t.Fatal("expected an error for a malformed number")
	}
}

// TestAnyNumberAndTypedNumberAreDistinct records the guarantee that the
// separate types exist for. It is a compile-time property, so the test can
// only state it: AnyNumber is not assignable to TypedNumber, which is what
// stops a lookup key reaching a write path. Uncommenting the line below is a
// compile error.
func TestAnyNumberAndTypedNumberAreDistinct(t *testing.T) {
	any := types.MustAnyNumber("12345")
	typed := types.MustTypedNumber("12345", types.TypeShortCode)

	// var _ types.TypedNumber = any   // compile error, by design

	if any.Unwrap() != typed.Unwrap() {
		t.Fatal("both wrap the same string; only their guarantees differ")
	}
}
