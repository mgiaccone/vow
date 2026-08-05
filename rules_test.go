package vow_test

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/mgiaccone/vow"
)

func TestNotBlank(t *testing.T) {
	if err := vow.NotBlank("hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err := vow.NotBlank("   ")
	if err == nil {
		t.Fatal("expected error for blank input")
	}
	if !errors.Is(err, vow.ErrBlank) {
		t.Fatalf("expected ErrBlank, got %v", err)
	}
	if err.Error() != "is required" {
		t.Fatalf("got message %q, want %q", err.Error(), "is required")
	}
}

func TestNotZero(t *testing.T) {
	if err := vow.NotZero("hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err := vow.NotZero("")
	if err == nil {
		t.Fatal("expected error for the zero value")
	}
	if !errors.Is(err, vow.ErrBlank) {
		t.Fatalf("expected ErrBlank, got %v", err)
	}
	if err.Error() != "is required" {
		t.Fatalf("got message %q, want %q", err.Error(), "is required")
	}
}

func TestNotZero_Int(t *testing.T) {
	if err := vow.NotZero(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := vow.NotZero(0); !errors.Is(err, vow.ErrBlank) {
		t.Fatalf("expected ErrBlank for 0, got %v", err)
	}
}

// TestNotZero_TimeTime is the reason NotZero exists: time.Time has no <
// operator, so it doesn't satisfy cmp.Ordered and Positive can't reject
// it. NotZero only needs comparable, which time.Time does satisfy.
func TestNotZero_TimeTime(t *testing.T) {
	if err := vow.NotZero(time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var zero time.Time
	if err := vow.NotZero(zero); !errors.Is(err, vow.ErrBlank) {
		t.Fatalf("expected ErrBlank for the zero time.Time, got %v", err)
	}
}

func TestMaxLen(t *testing.T) {
	rule := vow.MaxLen(3)
	if err := rule("abc"); err != nil {
		t.Fatalf("unexpected error at boundary: %v", err)
	}
	err := rule("abcd")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, vow.ErrTooLong) {
		t.Fatalf("expected ErrTooLong, got %v", err)
	}
	want := "must be at most 3 characters"
	if err.Error() != want {
		t.Fatalf("got message %q, want %q", err.Error(), want)
	}
}

func TestMaxLen_CountsRunesNotBytes(t *testing.T) {
	rule := vow.MaxLen(3)
	// "café" is 4 runes but 5 bytes with the 'é' encoded as 2 bytes.
	if err := rule("café"); err == nil {
		t.Fatal("expected error: café has 4 runes, over the limit of 3")
	}
	if err := rule("caf"); err != nil {
		t.Fatalf("unexpected error for 3-rune input: %v", err)
	}
}

func TestMinLen(t *testing.T) {
	rule := vow.MinLen(3)
	if err := rule("abc"); err != nil {
		t.Fatalf("unexpected error at boundary: %v", err)
	}
	err := rule("ab")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, vow.ErrTooShort) {
		t.Fatalf("expected ErrTooShort, got %v", err)
	}
	want := "must be at least 3 characters"
	if err.Error() != want {
		t.Fatalf("got message %q, want %q", err.Error(), want)
	}
}

func TestMatches(t *testing.T) {
	re := regexp.MustCompile(`^[a-z]+$`)
	rule := vow.Matches(re, "must be a valid email address")
	if err := rule("abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err := rule("ABC")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, vow.ErrNotMatch) {
		t.Fatalf("expected ErrNotMatch, got %v", err)
	}
	if err.Error() != "must be a valid email address" {
		t.Fatalf("got message %q", err.Error())
	}
}

func TestOneOf(t *testing.T) {
	rule := vow.OneOf("owner", "admin", "member")
	if err := rule("admin"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err := rule("wizard")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, vow.ErrNotInSet) {
		t.Fatalf("expected ErrNotInSet, got %v", err)
	}
	want := "must be one of: owner, admin, member"
	if err.Error() != want {
		t.Fatalf("got message %q, want %q", err.Error(), want)
	}
}

func TestInRange(t *testing.T) {
	rule := vow.InRange(1, 10)
	for _, v := range []int{1, 5, 10} {
		if err := rule(v); err != nil {
			t.Errorf("InRange(1,10)(%d) unexpected error: %v", v, err)
		}
	}
	err := rule(11)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, vow.ErrOutOfRange) {
		t.Fatalf("expected ErrOutOfRange, got %v", err)
	}
	want := "must be between 1 and 10"
	if err.Error() != want {
		t.Fatalf("got message %q, want %q", err.Error(), want)
	}
	if err := rule(0); err == nil {
		t.Fatal("expected error below range")
	}
}

func TestPositive(t *testing.T) {
	if err := vow.Positive(1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range []int{0, -1} {
		err := vow.Positive(v)
		if err == nil {
			t.Fatalf("expected error for %d", v)
		}
		if !errors.Is(err, vow.ErrOutOfRange) {
			t.Fatalf("expected ErrOutOfRange, got %v", err)
		}
		if err.Error() != "must be greater than zero" {
			t.Fatalf("got message %q", err.Error())
		}
	}
}

// TestNonNegative_AcceptsZero is the difference from Positive: a Balance
// that has hit exactly zero must still parse.
func TestNonNegative_AcceptsZero(t *testing.T) {
	for _, v := range []int{0, 1} {
		if err := vow.NonNegative(v); err != nil {
			t.Fatalf("unexpected error for %d: %v", v, err)
		}
	}
}

func TestNonNegative_RejectsNegative(t *testing.T) {
	err := vow.NonNegative(-1)
	if err == nil {
		t.Fatal("expected error for -1")
	}
	if !errors.Is(err, vow.ErrOutOfRange) {
		t.Fatalf("expected ErrOutOfRange, got %v", err)
	}
	want := "must not be negative"
	if err.Error() != want {
		t.Fatalf("got message %q, want %q", err.Error(), want)
	}
}
