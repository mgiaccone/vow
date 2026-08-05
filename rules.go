package vow

import (
	"cmp"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Sentinels for the built-in rules. Every reusable rule wraps one of these,
// so a caller can errors.Is a rejection reason even after it has been
// wrapped in a FieldError.
var (
	ErrBlank      = errors.New("is required")
	ErrTooLong    = errors.New("is too long")
	ErrTooShort   = errors.New("is too short")
	ErrNotMatch   = errors.New("has an invalid format")
	ErrNotInSet   = errors.New("is not an allowed value")
	ErrOutOfRange = errors.New("is out of range")
)

// ruleError pairs a transport-neutral message with the sentinel it stands
// for. Built-in rules return this instead of fmt.Errorf("...: %w", sentinel)
// so that a parameterised message like "must be at most 254 characters"
// stays a clean sentence fragment — wrapping would append the sentinel's own
// text ("is too long") and produce "must be at most 254 characters: is too
// long", which is not a fragment a UI can present after a field name.
// errors.Is still reaches the sentinel through Unwrap.
type ruleError struct {
	msg string
	err error
}

func (e ruleError) Error() string { return e.msg }
func (e ruleError) Unwrap() error { return e.err }

// NotBlank rejects a string that is empty once leading and trailing
// whitespace is removed.
func NotBlank(s string) error {
	if strings.TrimSpace(s) == "" {
		return ErrBlank
	}
	return nil
}

// MaxLen rejects a string longer than n runes.
func MaxLen(n int) Rule[string] {
	return func(s string) error {
		if utf8.RuneCountInString(s) > n {
			return ruleError{fmt.Sprintf("must be at most %d characters", n), ErrTooLong}
		}
		return nil
	}
}

// MinLen rejects a string shorter than n runes.
func MinLen(n int) Rule[string] {
	return func(s string) error {
		if utf8.RuneCountInString(s) < n {
			return ruleError{fmt.Sprintf("must be at least %d characters", n), ErrTooShort}
		}
		return nil
	}
}

// Matches rejects a string that does not match re, using msg as the
// rejection message.
func Matches(re *regexp.Regexp, msg string) Rule[string] {
	return func(s string) error {
		if !re.MatchString(s) {
			return ruleError{msg, ErrNotMatch}
		}
		return nil
	}
}

// OneOf rejects a string that is not one of allowed.
func OneOf(allowed ...string) Rule[string] {
	set := append([]string(nil), allowed...)
	msg := "must be one of: " + strings.Join(set, ", ")
	return func(s string) error {
		if !slicesContains(set, s) {
			return ruleError{msg, ErrNotInSet}
		}
		return nil
	}
}

// InRange rejects a value outside [lo, hi].
func InRange[T cmp.Ordered](lo, hi T) Rule[T] {
	return func(v T) error {
		if v < lo || v > hi {
			return ruleError{fmt.Sprintf("must be between %v and %v", lo, hi), ErrOutOfRange}
		}
		return nil
	}
}

// Positive rejects a value less than or equal to zero.
func Positive[T cmp.Ordered](v T) error {
	var zero T
	if v <= zero {
		return ruleError{"must be greater than zero", ErrOutOfRange}
	}
	return nil
}

// slicesContains avoids importing slices solely for this one call site with
// a comparable-but-not-Ordered constraint mismatch; string satisfies both,
// but keeping this local keeps the import list obviously minimal.
func slicesContains(set []string, s string) bool {
	for _, v := range set {
		if v == s {
			return true
		}
	}
	return false
}
