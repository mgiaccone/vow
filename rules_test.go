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

var errNotEven = errors.New("is not even")

func even(n int) error {
	if n%2 != 0 {
		return vow.Reject("must be even", errNotEven)
	}
	return nil
}

// TestReject_MessageExcludesSentinelText is the whole reason Reject exists.
// fmt.Errorf("must be even: %w", errNotEven) would render "must be even: is
// not even", which is not a fragment a UI can print after a field name.
func TestReject_MessageExcludesSentinelText(t *testing.T) {
	err := even(3)
	if err.Error() != "must be even" {
		t.Fatalf("got message %q, want %q", err.Error(), "must be even")
	}
	if !errors.Is(err, errNotEven) {
		t.Fatal("expected errors.Is to reach the sentinel")
	}
}

func TestReject_SentinelSurvivesFieldError(t *testing.T) {
	var c vow.Collector
	c.Add("Count", even(3))
	if !errors.Is(c.Err(), errNotEven) {
		t.Fatal("expected errors.Is to reach the sentinel through FieldError and Join")
	}
}

func TestWithMessage_ReplacesTextKeepsSentinel(t *testing.T) {
	r := vow.WithMessage(vow.MaxLen(3), "is too long for a code")
	err := r("abcd")
	if err == nil {
		t.Fatal("expected an error")
	}
	if err.Error() != "is too long for a code" {
		t.Fatalf("got message %q", err.Error())
	}
	if !errors.Is(err, vow.ErrTooLong) {
		t.Fatal("expected the rule's own sentinel to stay reachable")
	}
}

// TestWithMessage_OverBareSentinelRule covers the other rule shape: NotBlank
// returns ErrBlank itself rather than a ruleError, so the wrapping has to
// work on a bare sentinel too.
func TestWithMessage_OverBareSentinelRule(t *testing.T) {
	r := vow.WithMessage(vow.Rule[string](vow.NotBlank), "we need this")
	err := r("  ")
	if err.Error() != "we need this" {
		t.Fatalf("got message %q", err.Error())
	}
	if !errors.Is(err, vow.ErrBlank) {
		t.Fatal("expected ErrBlank to stay reachable")
	}
}

// TestWithSentinel_AddsWithoutReplacing is the decision this combinator turns
// on: a bad email is still genuinely a format failure, so a handler keyed on
// the general reason must keep matching alongside the specific one.
func TestWithSentinel_AddsWithoutReplacing(t *testing.T) {
	errBadEmail := errors.New("is not a valid email")
	pattern := regexp.MustCompile(`^[^@\s]+@[^@\s]+$`)
	r := vow.WithSentinel(vow.Matches(pattern, "must be a valid email address"), errBadEmail)

	err := r("nope")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !errors.Is(err, errBadEmail) {
		t.Fatal("expected the added sentinel to match")
	}
	if !errors.Is(err, vow.ErrNotMatch) {
		t.Fatal("expected the rule's original sentinel to still match")
	}
	if err.Error() != "must be a valid email address" {
		t.Fatalf("WithSentinel must not change the message, got %q", err.Error())
	}
}

func TestWithSentinel_OverBareSentinelRule(t *testing.T) {
	errRequired := errors.New("must be supplied")
	r := vow.WithSentinel(vow.Rule[string](vow.NotBlank), errRequired)
	err := r("")
	if !errors.Is(err, errRequired) || !errors.Is(err, vow.ErrBlank) {
		t.Fatalf("expected both sentinels to match, got %v", err)
	}
	if err.Error() != "is required" {
		t.Fatalf("got message %q", err.Error())
	}
}

// TestCombinators_ComposeEitherOrder: the custom message wins and every
// sentinel involved stays reachable, whichever way round they are applied.
func TestCombinators_ComposeEitherOrder(t *testing.T) {
	errMine := errors.New("is mine")

	outer := vow.WithSentinel(vow.WithMessage(vow.MaxLen(3), "custom"), errMine)
	inner := vow.WithMessage(vow.WithSentinel(vow.MaxLen(3), errMine), "custom")

	for name, r := range map[string]vow.Rule[string]{"sentinel(message)": outer, "message(sentinel)": inner} {
		err := r("abcd")
		if err.Error() != "custom" {
			t.Fatalf("%s: got message %q, want %q", name, err.Error(), "custom")
		}
		if !errors.Is(err, errMine) {
			t.Fatalf("%s: expected the added sentinel to match", name)
		}
		if !errors.Is(err, vow.ErrTooLong) {
			t.Fatalf("%s: expected ErrTooLong to stay reachable", name)
		}
	}
}

func TestCombinators_PassSuccessThrough(t *testing.T) {
	if err := vow.WithMessage(vow.MaxLen(10), "nope")("ok"); err != nil {
		t.Fatalf("WithMessage altered a success: %v", err)
	}
	if err := vow.WithSentinel(vow.MaxLen(10), errNotEven)("ok"); err != nil {
		t.Fatalf("WithSentinel altered a success: %v", err)
	}
}
