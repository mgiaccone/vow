package example_test

import (
	"fmt"
	"testing"

	"github.com/mgiaccone/vow"
	"github.com/mgiaccone/vow/example"
)

func ExampleNewRegisterNumber() {
	cmd, err := example.NewRegisterNumber("shortcode", "  12345  ")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(cmd.Type, cmd.Number)
	// Output: shortcode 12345
}

func ExampleNewRegisterNumber_dependentField() {
	_, err := example.NewRegisterNumber("lvn", "12345")
	for _, fe := range vow.FieldErrors(err) {
		fmt.Println(fe)
	}
	// Output: Number: must be a valid E.164 number
}

func TestNewRegisterNumber_Success(t *testing.T) {
	cmd, err := example.NewRegisterNumber("lvn", "+14155550123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Type != "lvn" || cmd.Number.Unwrap() != "+14155550123" {
		t.Fatalf("got %v / %v", cmd.Type, cmd.Number)
	}
}

// TestNewRegisterNumber_BadKindReportsOnce is what the c.OK guard buys: a
// bad kind must not also produce a fail-closed error against the number,
// which would be noise on top of the real complaint.
func TestNewRegisterNumber_BadKindReportsOnce(t *testing.T) {
	_, err := example.NewRegisterNumber("carrier-pigeon", "12345")
	if err == nil {
		t.Fatal("expected an error")
	}
	fes := vow.FieldErrors(err)
	if len(fes) != 1 {
		t.Fatalf("expected exactly 1 field error, got %d: %v", len(fes), fes)
	}
	if fes[0].Field != example.FieldNumberType {
		t.Fatalf("expected the error on %s, got %s", example.FieldNumberType, fes[0].Field)
	}
}

func TestNewRegisterNumber_BothFieldsBad(t *testing.T) {
	_, err := example.NewRegisterNumber("", "")
	fes := vow.FieldErrors(err)
	// The kind fails, so the number is never attempted: one error, not two.
	if len(fes) != 1 || fes[0].Field != example.FieldNumberType {
		t.Fatalf("got %v", fes)
	}
}

func TestNewLookupNumber(t *testing.T) {
	q, err := example.NewLookupNumber("+14155550123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Number.Unwrap() != "+14155550123" {
		t.Fatalf("got %q", q.Number.Unwrap())
	}

	if _, err := example.NewLookupNumber("nonsense"); err == nil {
		t.Fatal("expected an error for a malformed number")
	}
}
