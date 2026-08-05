package vow

import (
	"errors"
	"fmt"
)

// Field identifies a slot on a struct, e.g. "Inviter". Declare these as
// typed constants beside the struct that owns them so a typo is a compile
// error rather than a silently mislabeled failure.
type Field string

// FieldError attaches a Field to an underlying rule failure.
type FieldError struct {
	Field Field
	Err   error
}

// Error renders as "Inviter: must be a valid email address".
func (e FieldError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Err)
}

// Unwrap lets errors.Is and errors.As reach the sentinels declared in
// rules.go through a FieldError.
func (e FieldError) Unwrap() error {
	return e.Err
}

// Collector accumulates failures across several fields so a constructor can
// report all of them at once, instead of stopping at the first one the way
// Spec.Parse does for a single value.
type Collector struct {
	errs []FieldError
}

// Add records err against f. A nil err is ignored, so callers can pass a
// rule result straight through with no if statement:
//
//	c.Add(FieldExpiresAt, checkExpiry(expiresAt, now))
func (c *Collector) Add(f Field, err error) {
	if err == nil {
		return
	}
	c.errs = append(c.errs, FieldError{Field: f, Err: err})
}

// Err returns errors.Join of every FieldError recorded so far, or nil if
// none were.
func (c *Collector) Err() error {
	if len(c.errs) == 0 {
		return nil
	}
	errs := make([]error, len(c.errs))
	for i, e := range c.errs {
		errs[i] = e
	}
	return errors.Join(errs...)
}

// Collect runs parse against in and records any failure against f on c,
// returning parse's result. parse is any function shaped like a value
// object constructor — func(In) (Out, error) — so Collect works with
// generated constructors, handwritten ones, and constructors from other
// libraries alike; it does not depend on Spec.
//
// On failure Collect returns the zero value of Out; the caller must consult
// c.Err() before treating the result as usable. This matters for
// comparisons in particular: two values returned by failed calls to Collect
// both equal Out's zero value, so comparing two collected results with == or
// Equal without first checking c.Err() (or, for a generated type, IsZero)
// can manufacture a spurious cross-field match between two fields that both
// failed to parse.
func Collect[In, Out any](c *Collector, f Field, in In, parse func(In) (Out, error)) Out {
	out, err := parse(in)
	if err != nil {
		c.Add(f, err)
		var zero Out
		return zero
	}
	return out
}

// FieldErrors walks err — including trees built by errors.Join, and errors
// wrapped with fmt.Errorf's %w — and returns every FieldError found in it,
// in the order they were discovered. An empty result means err carries no
// field failures at all.
//
// FieldErrors intentionally does not use errors.As: on a joined error,
// errors.As stops at the first match and discards every other leaf, which
// would silently return one failure out of three from a Collector's error.
// Walking the tree by hand — matching FieldError directly, then
// interface{ Unwrap() []error } for joined errors, then
// interface{ Unwrap() error } for single-wrapped ones — is what makes
// FieldErrors return all of them.
func FieldErrors(err error) []FieldError {
	if err == nil {
		return nil
	}

	var out []FieldError
	var walk func(error)
	walk = func(e error) {
		if e == nil {
			return
		}
		switch fe := e.(type) {
		case FieldError:
			out = append(out, fe)
			// Descend into the wrapped error too, so a FieldError built
			// around a nested Collector's Err() still surfaces every leaf.
			walk(fe.Err)
			return
		case *FieldError:
			out = append(out, *fe)
			walk(fe.Err)
			return
		}
		if u, ok := e.(interface{ Unwrap() []error }); ok {
			for _, sub := range u.Unwrap() {
				walk(sub)
			}
			return
		}
		if u, ok := e.(interface{ Unwrap() error }); ok {
			walk(u.Unwrap())
		}
	}
	walk(err)
	return out
}
