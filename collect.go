package vow

import (
	"fmt"
	"strings"
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

// OK reports whether the named fields are free of recorded failures, so
// their collected values are safe to use. With no arguments it reports
// whether nothing at all has failed.
//
// This is the correct guard for logic that spans fields — a comparison
// between two of them, or a branch on one that decides how another is
// checked. Do not use IsZero for that: a type whose zero value is valid, an
// int with NonNegative say, reports IsZero() == true after a *successful*
// parse, so an IsZero guard skips checks that should run. A field never
// passed to Add or Collect counts as OK, having recorded no failure.
func (c *Collector) OK(fields ...Field) bool {
	if len(fields) == 0 {
		return len(c.errs) == 0
	}
	for _, e := range c.errs {
		for _, f := range fields {
			if e.Field == f {
				return false
			}
		}
	}
	return true
}

// CollectError is what Collector.Err returns: every field failure recorded,
// in the order they were recorded. It renders and unwraps exactly as the
// errors.Join it replaced, and exists so that callers have a type to name:
//
//	var ce vow.CollectError
//	if errors.As(err, &ce) {
//		// this is a validation failure, not a not-found or a bug
//	}
//
// Use the assertion to *classify* an error and FieldErrors to *extract* from
// it. errors.As stops at the first match, so if two command errors are ever
// joined together, ce holds only the first collector's fields while
// FieldErrors returns every field from both.
type CollectError []FieldError

// Error renders one field failure per line, matching what errors.Join
// produced before this type existed.
func (e CollectError) Error() string {
	parts := make([]string, len(e))
	for i, fe := range e {
		parts[i] = fe.Error()
	}
	return strings.Join(parts, "\n")
}

// Unwrap exposes the field failures to errors.Is and errors.As, so a rule
// sentinel stays reachable through the whole tree.
func (e CollectError) Unwrap() []error {
	errs := make([]error, len(e))
	for i, fe := range e {
		errs[i] = fe
	}
	return errs
}

// Err returns a CollectError holding every failure recorded so far, or nil if
// there were none.
//
// The return type is error rather than CollectError on purpose: returning a
// concrete type would put a nil CollectError inside a non-nil error interface
// at every call site, so `if err != nil` would fire even on success.
func (c *Collector) Err() error {
	if len(c.errs) == 0 {
		return nil
	}
	out := make(CollectError, len(c.errs))
	copy(out, c.errs)
	return out
}

// Collect runs parse against in and records any failure against f on c,
// returning parse's result. parse is any function shaped like a value
// object constructor — func(In) (Out, error) — so Collect works with
// generated constructors, handwritten ones, and constructors from other
// libraries alike; it does not depend on Spec.
//
// On failure Collect returns the zero value of Out; the caller must consult
// c.Err() before treating the result as usable. This matters for comparisons
// in particular: two values returned by failed calls to Collect both equal
// Out's zero value, so comparing two collected results without a guard can
// manufacture a spurious cross-field match between two fields that both
// failed to parse. Guard with Collector.OK, which asks whether those fields
// actually parsed — not with IsZero, which asks something different and gives
// the wrong answer for any type whose zero value is valid.
func Collect[In, Out any](c *Collector, f Field, in In, parse func(In) (Out, error)) Out {
	out, err := parse(in)
	if err != nil {
		c.Add(f, err)
		var zero Out
		return zero
	}
	return out
}

// CollectFunc runs parse and records any failure against f on c, returning
// parse's result. Unlike Collect, parse takes no argument: it closes over
// whatever it needs, which is what makes it fit a constructor taking more
// than the value alone — one whose spec takes parameters, for instance.
//
//	cmd.PostalCode = vow.CollectFunc(&c, FieldPostalCode, func() (types.PostalCode, error) {
//		return types.NewPostalCode(raw, cmd.Country)
//	})
//
// Prefer this to calling the constructor yourself and passing the error to
// Add: a bare c.Add(f, err) reads as a statement that silently does nothing
// when err is nil, which is the shape Go's explicit error handling exists to
// avoid.
//
// On failure CollectFunc returns the zero value of Out, exactly as Collect
// does; guard with Collector.OK before using the result.
func CollectFunc[Out any](c *Collector, f Field, parse func() (Out, error)) Out {
	out, err := parse()
	if err != nil {
		c.Add(f, err)
		var zero Out
		return zero
	}
	return out
}

// ElementError pairs the index of a slice element with the failure that
// element's constructor returned.
type ElementError struct {
	Index int
	Err   error
}

// ElementErrors is the error a slice field carries when one or more of its
// elements failed to parse. CollectSlice records exactly one of these per
// field, rather than one FieldError per bad element, so that Field keeps
// naming the field and nothing else — Collector.OK, and any map keyed by
// Field, go on working unchanged.
//
// It also keeps the true element indices. Recording each failure separately
// would let a consumer grouping by field number them only by their order
// among failures, reporting 0 and 1 for what were really elements 1 and 3.
type ElementErrors []ElementError

// Error renders every element failure on one line, each prefixed with its
// index: "[1] must be a valid email address; [3] is required". A consumer
// that wants them individually should use errors.As to recover the
// ElementErrors and read Index off each entry.
func (e ElementErrors) Error() string {
	parts := make([]string, len(e))
	for i, el := range e {
		parts[i] = fmt.Sprintf("[%d] %s", el.Index, el.Err)
	}
	return strings.Join(parts, "; ")
}

// Unwrap exposes the element failures to errors.Is and errors.As, so a rule
// sentinel stays reachable from the joined error a Collector returns.
func (e ElementErrors) Unwrap() []error {
	errs := make([]error, len(e))
	for i, el := range e {
		errs[i] = el.Err
	}
	return errs
}

// Element pairs a parsed value with the index of the input it came from.
// Options receive these rather than bare values so that a failure they report
// names the caller's position, which stays correct even when earlier elements
// failed to parse and are absent from the slice.
type Element[Out any] struct {
	Index int
	Value Out
}

// SliceOption adjusts or checks what CollectSlice parsed. Options run in
// order, after every element has been attempted, and apply to the collection
// rather than to any one element. An option that records a failure makes
// CollectSlice return nil, exactly as a failing element does.
//
// Options run even when some elements failed to parse, over the ones that
// succeeded, so a duplicate is not hidden behind an unrelated bad element and
// discovered only on the next attempt. This is why they take []Element rather
// than []Out: the surviving values are compacted, so their position in the
// slice is no longer their position in the caller's input, and only Element
// still knows the difference.
//
// Pass one as the function itself rather than calling it — vow.Deduped, not
// vow.Deduped() — which is what lets Go infer the element type from the
// constructor instead of making you name it at every call site. It is also
// why the built-in options take no parameters.
type SliceOption[Out any] func(c *Collector, f Field, s []Element[Out]) []Element[Out]

// NotEmpty records a failure when the list has no elements:
//
//	cmd.Recipients = vow.CollectSlice(&c, FieldRecipients, raw, types.NewEmail, vow.NotEmpty)
//
// Without it an empty list parses cleanly, because CollectSlice reports on the
// elements it was given and an absent element cannot fail. Whether "none" is
// acceptable belongs to the field, not to the helper, which is why it is an
// option rather than the default.
//
// It stays quiet when the field has already failed, so a list whose elements
// were all rejected reports those rejections rather than also claiming to be
// empty — which it was not.
//
// It reuses ErrBlank, the sentinel NotBlank and NotZero return: a required
// list with nothing in it is missing in the same sense a required string with
// nothing in it is. Unlike the deduplication options it places no constraint
// on the element type.
func NotEmpty[T any](c *Collector, f Field, s []Element[T]) []Element[T] {
	if len(s) == 0 && c.OK(f) {
		c.Add(f, Reject("must not be empty", ErrBlank))
	}
	return s
}

// Deduped drops later occurrences of a value that already appeared, keeping
// the order of first appearance:
//
//	cmd.Tags = vow.CollectSlice(&c, FieldTags, raw, types.NewTag, vow.Deduped)
//
// It records nothing: duplicates are normalized away rather than reported.
// This is the one place vow discards what the caller sent without saying so,
// which is the point of it — reach for NoDuplicates when a repeat is a
// mistake worth surfacing instead.
//
// Deduplicating parsed values rather than raw input is deliberate and is why
// this runs where it does. "  A@Example.com " and "a@example.com" are
// different strings but the same Email once sanitize=trim|lower has run, so
// deduplicating the input would miss the pair entirely.
func Deduped[T comparable](c *Collector, f Field, s []Element[T]) []Element[T] {
	seen := make(map[T]struct{}, len(s))
	out := s[:0:0]
	for _, el := range s {
		if _, dup := seen[el.Value]; dup {
			continue
		}
		seen[el.Value] = struct{}{}
		out = append(out, el)
	}
	return out
}

// NoDuplicates records a failure for every repeated element instead of
// removing it, naming the earlier element it duplicates:
//
//	cmd.Recipients = vow.CollectSlice(&c, FieldRecipients, raw, types.NewEmail, vow.NoDuplicates)
//
// Failures arrive as an ElementErrors carrying the caller's own indices, so
// they read and unpack exactly like element parse failures — "[3] duplicates
// item 1" — and errors.Is reaches ErrDuplicate.
//
// Deduped and NoDuplicates are alternatives, not a pair to combine: passing
// both asks to report duplicates and silently remove them at once.
func NoDuplicates[T comparable](c *Collector, f Field, s []Element[T]) []Element[T] {
	seen := make(map[T]int, len(s))
	var dupes ElementErrors
	for _, el := range s {
		if first, dup := seen[el.Value]; dup {
			dupes = append(dupes, ElementError{
				Index: el.Index,
				Err:   Reject(fmt.Sprintf("duplicates item %d", first), ErrDuplicate),
			})
			continue
		}
		seen[el.Value] = el.Index
	}
	if len(dupes) > 0 {
		c.Add(f, dupes)
	}
	return s
}

// CollectSlice runs parse against every element of in and records the
// failures against f on c, returning the parsed elements. parse is the same
// func(In) (Out, error) that Collect takes, so a value object constructor
// drops straight in:
//
//	cmd.Recipients = vow.CollectSlice(&c, FieldRecipients, raw, types.NewEmail)
//
// A constructor needing more than the value closes over the rest, exactly as
// with CollectFunc, so there is no CollectSlice variant for that case:
//
//	cmd.Codes = vow.CollectSlice(&c, FieldCodes, raw, func(s string) (types.PostalCode, error) {
//		return types.NewPostalCode(s, cmd.Country)
//	})
//
// Every element is attempted. Spec.Parse stops at the first failing rule
// because that concerns a single value, but a slice is many values, and
// reporting all of them at once is what Collector exists for — so a caller
// fixing a bad list learns about every bad element in one round-trip.
//
// Element parse failures arrive as one FieldError holding an ElementErrors,
// rather than one entry per bad element, so Field goes on naming the field
// and nothing else. An option that reports adds its own entry, so a list with
// both a bad element and a duplicate produces two — each carrying the true
// indices.
//
// On any failure CollectSlice returns nil rather than the elements that did
// parse: a short slice would let a caller who forgot c.Err() act on a subset,
// which is the quiet failure this helper exists to prevent.
//
// Options run after every element has been attempted, and apply to the
// collection rather than to any one element — see NotEmpty, Deduped, and
// NoDuplicates. They run even when some elements failed, over the ones that
// succeeded, so a duplicate is reported in the same round as the bad element
// beside it instead of surfacing only once that is fixed. An option that
// records a failure makes CollectSlice return nil too, so the contract is the
// same whichever way the field failed.
//
// An empty or nil in records nothing on its own: there are no elements to
// fail. Whether a list is allowed to be empty is the field's business rather
// than this helper's, so pass NotEmpty when it is not.
func CollectSlice[In, Out any](c *Collector, f Field, in []In, parse func(In) (Out, error), opts ...SliceOption[Out]) []Out {
	kept := make([]Element[Out], 0, len(in))
	var bad ElementErrors
	for i, v := range in {
		item, err := parse(v)
		if err != nil {
			bad = append(bad, ElementError{Index: i, Err: err})
			continue
		}
		kept = append(kept, Element[Out]{Index: i, Value: item})
	}
	// Recorded before the options run, so an option can see that the field
	// has already failed — NotEmpty relies on it to stay quiet when the list
	// was full of bad values rather than actually empty.
	if len(bad) > 0 {
		c.Add(f, bad)
	}

	// Compare against a snapshot rather than c.OK(f): the caller may already
	// have recorded a failure for f before calling, and that is not this
	// call's to report on.
	before := len(c.errs)
	for _, opt := range opts {
		kept = opt(c, f, kept)
	}
	if len(bad) > 0 || len(c.errs) > before {
		return nil
	}

	out := make([]Out, len(kept))
	for i, el := range kept {
		out[i] = el.Value
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
