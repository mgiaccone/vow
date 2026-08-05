// Package vow generates value objects and enums for Go: types whose
// underlying value can only be constructed through a fallible, sanitizing
// constructor. The rule that governs every decision in this package is
// Validated On Write — a value is parsed once, at construction and is
// unforgeable thereafter.
package vow

// Sanitizer normalizes a value. It cannot fail: anything that can fail is a
// Rule, not a Sanitizer. That distinction keeps the set of sanitizers closed
// enough to name safely in a struct tag — see Trim, Lower, Upper, and
// Collapse.
type Sanitizer[T any] func(T) T

// Rule reports whether a value is acceptable. A non-nil error means the
// value was rejected; its text should read as a sentence fragment following
// a field name, e.g. "must be at most 254 characters", never a full sentence
// or a transport-specific message.
type Rule[T any] func(T) error

// Spec describes how to sanitize and validate a value of type T. The zero
// Spec sanitizes with the identity function and accepts every value.
type Spec[T any] struct {
	// Sanitize normalizes the input before Rules run. It may be nil.
	Sanitize Sanitizer[T]

	// Rules run in order against the sanitized value. Parse returns the
	// first failure and stops; it does not run the remaining rules.
	Rules []Rule[T]
}

// Parse sanitizes in, then runs Rules in order against the sanitized value,
// returning on the first rule that fails.
//
// Parse is deliberately fail-fast: one value, one reason. Aggregating
// failures across several values — the fields of a command, say — is the
// job of Collector and Collect, not Spec. On failure Parse returns the zero
// value of T, not the sanitized input, so a rejected value can never leak
// into use.
func (s Spec[T]) Parse(in T) (T, error) {
	v := in
	if s.Sanitize != nil {
		v = s.Sanitize(v)
	}
	for _, rule := range s.Rules {
		if rule == nil {
			continue
		}
		if err := rule(v); err != nil {
			var zero T
			return zero, err
		}
	}
	return v, nil
}

// Sanitizing returns a copy of s whose Sanitize is Chain(ss...), replacing
// (not composing with) any existing Sanitize. The generator calls this to
// apply tag-declared sanitizers to a spec, so a handwritten Spec meant to
// be used with sanitize= in a tag should leave Sanitize nil — Sanitizing is
// what attaches it.
func (s Spec[T]) Sanitizing(ss ...Sanitizer[T]) Spec[T] {
	s.Sanitize = Chain(ss...)
	return s
}

// Chain composes sanitizers into one, applying them left to right. Nil
// sanitizers are skipped. Chain() returns the identity sanitizer.
func Chain[T any](ss ...Sanitizer[T]) Sanitizer[T] {
	fns := make([]Sanitizer[T], 0, len(ss))
	for _, fn := range ss {
		if fn != nil {
			fns = append(fns, fn)
		}
	}
	return func(v T) T {
		for _, fn := range fns {
			v = fn(v)
		}
		return v
	}
}
