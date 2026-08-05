package vow_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mgiaccone/vow"
)

func TestSpecParse_SanitizeThenValidate(t *testing.T) {
	spec := vow.Spec[string]{
		Sanitize: vow.Trim,
		Rules:    []vow.Rule[string]{vow.NotBlank},
	}

	got, err := spec.Parse("   ")
	if err == nil {
		t.Fatalf("expected error for blank-after-trim input, got value %q", got)
	}
	if !errors.Is(err, vow.ErrBlank) {
		t.Fatalf("expected ErrBlank, got %v", err)
	}
	if got != "" {
		t.Fatalf("expected zero value on failure, got %q", got)
	}

	got, err = spec.Parse("  hello  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("expected sanitized value %q, got %q", "hello", got)
	}
}

func TestSpecParse_FailFastOnFirstRule(t *testing.T) {
	var secondRan bool
	spec := vow.Spec[string]{
		Rules: []vow.Rule[string]{
			func(string) error { return vow.ErrBlank },
			func(string) error { secondRan = true; return nil },
		},
	}
	if _, err := spec.Parse("x"); !errors.Is(err, vow.ErrBlank) {
		t.Fatalf("expected ErrBlank, got %v", err)
	}
	if secondRan {
		t.Fatal("second rule ran after the first one failed; Parse must fail fast")
	}
}

func TestSpecParse_NilSanitizeAndNilRules(t *testing.T) {
	var spec vow.Spec[string]
	got, err := spec.Parse("hello")
	if err != nil {
		t.Fatalf("zero Spec must accept everything, got err %v", err)
	}
	if got != "hello" {
		t.Fatalf("zero Spec must not alter the value, got %q", got)
	}
}

func TestSpecParse_SkipsNilRule(t *testing.T) {
	spec := vow.Spec[string]{Rules: []vow.Rule[string]{nil}}
	if _, err := spec.Parse("hello"); err != nil {
		t.Fatalf("nil rule must be skipped, got err %v", err)
	}
}

func TestSpecSanitizing_ReplacesNotComposes(t *testing.T) {
	base := vow.Spec[string]{Sanitize: vow.Upper}
	replaced := base.Sanitizing(vow.Trim)

	got, err := replaced.Parse("  hello  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("Sanitizing must replace the existing sanitizer, got %q", got)
	}

	// The original Spec must be unaffected: Sanitizing returns a copy.
	got2, err := base.Parse("  hello  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got2 != "  HELLO  " {
		t.Fatalf("original Spec must be unchanged, got %q", got2)
	}
}

func TestChain_OrderingAndEmpty(t *testing.T) {
	order := vow.Chain(vow.Trim, vow.Lower)
	if got := order("  ABC  "); got != "abc" {
		t.Fatalf("expected left-to-right composition, got %q", got)
	}

	identity := vow.Chain[string]()
	if got := identity("  ABC  "); got != "  ABC  " {
		t.Fatalf("Chain() must be the identity sanitizer, got %q", got)
	}
}

func TestChain_SkipsNil(t *testing.T) {
	fn := vow.Chain[string](nil, vow.Upper, nil)
	if got := fn("abc"); got != "ABC" {
		t.Fatalf("Chain must skip nil sanitizers, got %q", got)
	}
}

func TestExampleUsage_SanitizeThenLower(t *testing.T) {
	spec := vow.Spec[string]{}.Sanitizing(vow.Trim, vow.Lower)
	got, err := spec.Parse("  ADMIN  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.EqualFold(got, "admin") || got != "admin" {
		t.Fatalf("expected %q, got %q", "admin", got)
	}
}
