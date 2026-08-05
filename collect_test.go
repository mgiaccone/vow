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
