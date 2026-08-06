package vow_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/mgiaccone/vow"
)

const (
	fieldInviter   vow.Field = "Inviter"
	fieldInvitee   vow.Field = "Invitee"
	fieldRole      vow.Field = "Role"
	fieldExpiresAt vow.Field = "ExpiresAt"
)

func parseNonBlank(s string) (string, error) {
	if err := vow.NotBlank(s); err != nil {
		return "", err
	}
	return s, nil
}

func TestCollector_Empty_ErrIsNil(t *testing.T) {
	var c vow.Collector
	if err := c.Err(); err != nil {
		t.Fatalf("expected nil for an empty Collector, got %v", err)
	}
}

func TestCollector_AddIgnoresNil(t *testing.T) {
	var c vow.Collector
	c.Add(fieldInviter, nil)
	if err := c.Err(); err != nil {
		t.Fatalf("Add(nil) must not record a failure, got %v", err)
	}
}

func TestCollectorOK_NoArguments(t *testing.T) {
	var c vow.Collector
	if !c.OK() {
		t.Fatal("an empty Collector must report OK")
	}
	c.Add(fieldInviter, vow.ErrBlank)
	if c.OK() {
		t.Fatal("OK() must be false once anything has failed")
	}
}

func TestCollectorOK_NamedFields(t *testing.T) {
	var c vow.Collector
	c.Add(fieldInviter, vow.ErrBlank)

	if c.OK(fieldInviter) {
		t.Error("OK must be false for a field that failed")
	}
	if !c.OK(fieldInvitee) {
		t.Error("OK must be true for a field that did not fail")
	}
	if !c.OK(fieldRole) {
		t.Error("OK must be true for a field never passed to Add or Collect")
	}
	if c.OK(fieldInvitee, fieldInviter) {
		t.Error("OK must be false when any of the named fields failed")
	}
	if !c.OK(fieldInvitee, fieldRole) {
		t.Error("OK must be true when none of the named fields failed")
	}
}

// TestCollectorOK_ZeroValueThatParsed is the reason OK exists. Guarding
// cross-field logic with IsZero is only sound when a type's zero value can
// never be valid. Here 0 parses successfully under NonNegative and is also
// the zero value, so an IsZero guard would skip a check that should run,
// while OK correctly reports the field as parsed.
func TestCollectorOK_ZeroValueThatParsed(t *testing.T) {
	spec := vow.Spec[int]{Rules: []vow.Rule[int]{vow.NonNegative[int]}}

	var c vow.Collector
	got := vow.Collect(&c, fieldRole, 0, spec.Parse)

	if got != 0 {
		t.Fatalf("expected 0 to parse successfully, got %d", got)
	}
	if !c.OK(fieldRole) {
		t.Fatal("a successful parse returning the zero value must report OK")
	}
	// The distinction the guard turns on: the value *is* the zero value,
	// which is exactly why IsZero is the wrong question to ask.
	if got != *new(int) {
		t.Fatal("expected the parsed value to equal the zero value")
	}
}

func TestCollectFunc_Success(t *testing.T) {
	var c vow.Collector
	got := vow.CollectFunc(&c, fieldInviter, func() (string, error) {
		return "a@example.com", nil
	})
	if got != "a@example.com" {
		t.Fatalf("got %q", got)
	}
	if err := c.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.OK(fieldInviter) {
		t.Fatal("expected the field to report OK")
	}
}

func TestCollectFunc_Failure(t *testing.T) {
	var c vow.Collector
	got := vow.CollectFunc(&c, fieldInviter, func() (string, error) {
		return "ignored", vow.ErrBlank
	})
	if got != "" {
		t.Fatalf("expected the zero value on failure, got %q", got)
	}
	if c.OK(fieldInviter) {
		t.Fatal("expected the field to report not OK")
	}
	if !errors.Is(c.Err(), vow.ErrBlank) {
		t.Fatalf("expected ErrBlank through the collector, got %v", c.Err())
	}
}

// TestCollectFunc_ClosesOverSeveralArguments is the case CollectFunc exists
// for: a constructor taking more than the value cannot satisfy Collect's
// func(In) (Out, error), but a thunk closing over the extras can.
func TestCollectFunc_ClosesOverSeveralArguments(t *testing.T) {
	newSuffixed := func(prefix, sep, in string) (string, error) {
		if in == "" {
			return "", vow.ErrBlank
		}
		return prefix + sep + in, nil
	}

	var c vow.Collector
	got := vow.CollectFunc(&c, fieldRole, func() (string, error) {
		return newSuffixed("role", ":", "admin")
	})
	if got != "role:admin" {
		t.Fatalf("got %q", got)
	}
	if err := c.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCollect_ZeroValueOnFailure(t *testing.T) {
	var c vow.Collector
	got := vow.Collect(&c, fieldInviter, "   ", parseNonBlank)
	if got != "" {
		t.Fatalf("expected zero value on failure, got %q", got)
	}
	if err := c.Err(); err == nil {
		t.Fatal("expected Collect to record the failure against the field")
	}
}

func TestCollect_PassesThroughOnSuccess(t *testing.T) {
	var c vow.Collector
	got := vow.Collect(&c, fieldInviter, "a@example.com", parseNonBlank)
	if got != "a@example.com" {
		t.Fatalf("got %q, want %q", got, "a@example.com")
	}
	if err := c.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCollector_AggregatesThreeFailures(t *testing.T) {
	var c vow.Collector
	vow.Collect(&c, fieldInviter, "", parseNonBlank)
	vow.Collect(&c, fieldInvitee, "", parseNonBlank)
	c.Add(fieldRole, vow.OneOf("owner", "admin", "member")("wizard"))

	err := c.Err()
	if err == nil {
		t.Fatal("expected an aggregated error")
	}
	fes := vow.FieldErrors(err)
	if len(fes) != 3 {
		t.Fatalf("expected 3 field errors, got %d: %v", len(fes), fes)
	}

	got := map[vow.Field]bool{}
	for _, fe := range fes {
		got[fe.Field] = true
	}
	for _, f := range []vow.Field{fieldInviter, fieldInvitee, fieldRole} {
		if !got[f] {
			t.Errorf("expected a FieldError for %s, got none", f)
		}
	}
}

func TestFieldError_ErrorsIsReachesSentinel(t *testing.T) {
	fe := vow.FieldError{Field: fieldInviter, Err: vow.ErrBlank}
	if !errors.Is(fe, vow.ErrBlank) {
		t.Fatal("expected errors.Is to reach ErrBlank through FieldError.Unwrap")
	}
}

func TestFieldError_ErrorsIsReachesSentinelThroughJoin(t *testing.T) {
	var c vow.Collector
	c.Add(fieldInviter, vow.ErrBlank)
	c.Add(fieldRole, vow.ErrNotInSet)
	err := c.Err()

	if !errors.Is(err, vow.ErrBlank) {
		t.Fatal("expected errors.Is to reach ErrBlank through the joined error")
	}
	if !errors.Is(err, vow.ErrNotInSet) {
		t.Fatal("expected errors.Is to reach ErrNotInSet through the joined error")
	}
}

// TestErrorsAs_FindsOnlyOneLeaf is the reason FieldErrors exists: on a
// joined error, errors.As stops at the first match and silently discards
// the rest, so a caller using errors.As alone would learn about only one of
// three failed fields. FieldErrors exists to walk every leaf instead.
func TestErrorsAs_FindsOnlyOneLeaf(t *testing.T) {
	var c vow.Collector
	c.Add(fieldInviter, vow.ErrBlank)
	c.Add(fieldInvitee, vow.ErrBlank)
	c.Add(fieldRole, vow.ErrNotInSet)
	err := c.Err()

	var fe vow.FieldError
	if !errors.As(err, &fe) {
		t.Fatal("expected errors.As to find at least one FieldError")
	}

	all := vow.FieldErrors(err)
	if len(all) != 3 {
		t.Fatalf("errors.As found only %+v, but FieldErrors correctly found %d leaves: %v", fe, len(all), all)
	}
}

func TestFieldErrors_NonFieldError(t *testing.T) {
	err := errors.New("plain error, not a field failure")
	if fes := vow.FieldErrors(err); len(fes) != 0 {
		t.Fatalf("expected no field errors for a plain error, got %v", fes)
	}
}

func TestFieldErrors_Nil(t *testing.T) {
	if fes := vow.FieldErrors(nil); len(fes) != 0 {
		t.Fatalf("expected no field errors for nil, got %v", fes)
	}
}

func TestFieldErrors_WrappedWithFmtErrorf(t *testing.T) {
	fe := vow.FieldError{Field: fieldExpiresAt, Err: vow.ErrOutOfRange}
	wrapped := fmt.Errorf("creating invite: %w", fe)

	got := vow.FieldErrors(wrapped)
	if len(got) != 1 {
		t.Fatalf("expected 1 field error through fmt.Errorf wrapping, got %d", len(got))
	}
	if got[0].Field != fieldExpiresAt {
		t.Fatalf("got field %s, want %s", got[0].Field, fieldExpiresAt)
	}
}

func TestFieldError_ErrorString(t *testing.T) {
	fe := vow.FieldError{Field: fieldInviter, Err: errors.New("must be a valid email address")}
	want := "Inviter: must be a valid email address"
	if fe.Error() != want {
		t.Fatalf("got %q, want %q", fe.Error(), want)
	}
	if !strings.Contains(fe.Error(), string(fieldInviter)) {
		t.Fatalf("expected field name in error string, got %q", fe.Error())
	}
}
