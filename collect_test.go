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

const fieldRecipients vow.Field = "Recipients"

func TestCollectSlice_AllValid(t *testing.T) {
	var c vow.Collector
	out := vow.CollectSlice(&c, fieldRecipients, []string{"a", "b", "c"}, parseNonBlank)

	if err := c.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 3 || out[0] != "a" || out[2] != "c" {
		t.Fatalf("got %v, want [a b c] in order", out)
	}
}

func TestCollectSlice_OneInvalid_RecordsExactlyOneEntry(t *testing.T) {
	var c vow.Collector
	out := vow.CollectSlice(&c, fieldRecipients, []string{"a", "", "c"}, parseNonBlank)

	if out != nil {
		t.Fatalf("expected nil on failure, got %v", out)
	}
	fes := vow.FieldErrors(c.Err())
	if len(fes) != 1 {
		t.Fatalf("expected 1 FieldError for the field, got %d: %v", len(fes), fes)
	}
	if fes[0].Field != fieldRecipients {
		t.Fatalf("got field %q", fes[0].Field)
	}
}

// TestCollectSlice_KeepsTrueIndices is the behavior the whole design turns
// on. Elements 1 and 3 of four fail, and they must be reported as 1 and 3 —
// not 0 and 1, which is what numbering failures by their own order would give.
func TestCollectSlice_KeepsTrueIndices(t *testing.T) {
	var c vow.Collector
	vow.CollectSlice(&c, fieldRecipients, []string{"a", "", "c", ""}, parseNonBlank)

	var ee vow.ElementErrors
	if !errors.As(vow.FieldErrors(c.Err())[0].Err, &ee) {
		t.Fatal("expected the FieldError to carry an ElementErrors")
	}
	if len(ee) != 2 {
		t.Fatalf("expected 2 element failures, got %d", len(ee))
	}
	if ee[0].Index != 1 || ee[1].Index != 3 {
		t.Fatalf("got indices %d and %d, want 1 and 3", ee[0].Index, ee[1].Index)
	}
}

// TestCollectSlice_AttemptsEveryElement guards a promise to the caller: a bad
// element must not stop the ones after it from being checked.
func TestCollectSlice_AttemptsEveryElement(t *testing.T) {
	var c vow.Collector
	calls := 0
	parse := func(s string) (string, error) {
		calls++
		return parseNonBlank(s)
	}

	vow.CollectSlice(&c, fieldRecipients, []string{"", "", ""}, parse)
	if calls != 3 {
		t.Fatalf("parse called %d times, want 3 — it stopped early", calls)
	}
}

// TestCollectSlice_OKIsFalse is what keeps Field plain rather than indexed:
// an indexed field would leave this guard reporting true.
func TestCollectSlice_OKIsFalse(t *testing.T) {
	var c vow.Collector
	vow.CollectSlice(&c, fieldRecipients, []string{"a", "", "c", ""}, parseNonBlank)

	if c.OK(fieldRecipients) {
		t.Fatal("expected OK to report false while elements are invalid")
	}
	if c.OK() {
		t.Fatal("expected the no-argument form to report false too")
	}
}

// TestCollectSlice_ErrorsIsReachesLaterElement walks the whole chain —
// Join, FieldError, ElementErrors, element error — and does it for an element
// that is not the first, so a shortcut that only unwrapped one would fail.
func TestCollectSlice_ErrorsIsReachesLaterElement(t *testing.T) {
	var c vow.Collector
	parse := func(s string) (string, error) {
		if s == "toolong" {
			return "", vow.MaxLen(3)(s)
		}
		return parseNonBlank(s)
	}

	vow.CollectSlice(&c, fieldRecipients, []string{"a", "b", "toolong"}, parse)
	if !errors.Is(c.Err(), vow.ErrTooLong) {
		t.Fatal("expected errors.Is to reach a later element's sentinel")
	}
}

// TestCollectSlice_UntypedConsumerLosesNothing: a caller that never heard of
// ElementErrors still gets every failure, just rendered on one line.
func TestCollectSlice_UntypedConsumerLosesNothing(t *testing.T) {
	var c vow.Collector
	vow.CollectSlice(&c, fieldRecipients, []string{"a", "", "c", ""}, parseNonBlank)

	msg := vow.FieldErrors(c.Err())[0].Err.Error()
	if !strings.Contains(msg, "[1]") || !strings.Contains(msg, "[3]") {
		t.Fatalf("expected both indices in the rendered message, got %q", msg)
	}
}

func TestCollectSlice_NilAndEmptyRecordNothing(t *testing.T) {
	var c vow.Collector
	if out := vow.CollectSlice(&c, fieldRecipients, nil, parseNonBlank); len(out) != 0 {
		t.Fatalf("expected empty result for nil input, got %v", out)
	}
	if out := vow.CollectSlice(&c, fieldRecipients, []string{}, parseNonBlank); len(out) != 0 {
		t.Fatalf("expected empty result for empty input, got %v", out)
	}
	if err := c.Err(); err != nil {
		t.Fatalf("expected no failures recorded, got %v", err)
	}
}

// TestCollectSlice_ClosesOverDiscriminator is why there is no CollectSliceFunc:
// parse is a func value, so a constructor needing more than the element closes
// over the rest.
func TestCollectSlice_ClosesOverDiscriminator(t *testing.T) {
	var c vow.Collector
	limit := 3

	out := vow.CollectSlice(&c, fieldRecipients, []string{"ab", "cd"}, func(s string) (string, error) {
		if err := vow.MaxLen(limit)(s); err != nil {
			return "", err
		}
		return s, nil
	})

	if err := c.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("got %v", out)
	}
}

const fieldTags vow.Field = "Tags"

func TestDeduped_KeepsFirstAppearanceOrder(t *testing.T) {
	var c vow.Collector
	out := vow.CollectSlice(&c, fieldTags, []string{"b", "a", "b", "c", "a"}, parseNonBlank, vow.Deduped)

	if err := c.Err(); err != nil {
		t.Fatalf("Deduped must record nothing, got %v", err)
	}
	want := []string{"b", "a", "c"}
	if len(out) != len(want) {
		t.Fatalf("got %v, want %v", out, want)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("got %v, want %v", out, want)
		}
	}
}

// TestDeduped_DoesNotClobberInput: Deduped filters in place over a fresh
// backing array, so a caller's own slice must survive being passed to it.
func TestDeduped_DoesNotClobberInput(t *testing.T) {
	var c vow.Collector
	in := []string{"a", "a", "b"}
	vow.Deduped(&c, fieldTags, in)

	if in[0] != "a" || in[1] != "a" || in[2] != "b" {
		t.Fatalf("input was modified: %v", in)
	}
}

// TestNoDuplicates_ReportsWithTrueIndices names the earlier element, and uses
// the same ElementErrors path as a parse failure so consumers need no special
// case.
func TestNoDuplicates_ReportsWithTrueIndices(t *testing.T) {
	var c vow.Collector
	out := vow.CollectSlice(&c, fieldTags, []string{"a", "b", "a", "b"}, parseNonBlank, vow.NoDuplicates)

	if out != nil {
		t.Fatalf("expected nil when an option rejects, got %v", out)
	}
	if c.OK(fieldTags) {
		t.Fatal("expected OK to be false")
	}

	var ee vow.ElementErrors
	if !errors.As(vow.FieldErrors(c.Err())[0].Err, &ee) {
		t.Fatal("expected duplicates to arrive as ElementErrors")
	}
	if len(ee) != 2 || ee[0].Index != 2 || ee[1].Index != 3 {
		t.Fatalf("got %v, want repeats at indices 2 and 3", ee)
	}
	if ee[0].Err.Error() != "duplicates item 0" {
		t.Fatalf("got %q", ee[0].Err.Error())
	}
	if !errors.Is(c.Err(), vow.ErrDuplicate) {
		t.Fatal("expected errors.Is to reach ErrDuplicate")
	}
}

func TestNoDuplicates_SilentWhenUnique(t *testing.T) {
	var c vow.Collector
	out := vow.CollectSlice(&c, fieldTags, []string{"a", "b", "c"}, parseNonBlank, vow.NoDuplicates)

	if err := c.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("got %v", out)
	}
}

// TestSliceOptions_SkippedWhenAnElementFailed: options describe the
// collection, so running them over a list that never fully parsed would
// report on values the caller never successfully supplied.
func TestSliceOptions_SkippedWhenAnElementFailed(t *testing.T) {
	var c vow.Collector
	ran := false
	spy := func(c *vow.Collector, f vow.Field, s []string) []string {
		ran = true
		return s
	}

	vow.CollectSlice(&c, fieldTags, []string{"a", ""}, parseNonBlank, spy)
	if ran {
		t.Fatal("expected options to be skipped once an element failed")
	}
}

// TestSliceOptions_PreexistingFieldErrorDoesNotForceNil guards the snapshot:
// CollectSlice compares against the error count it saw on entry, so a failure
// the caller recorded earlier is not mistaken for one of its own options.
func TestSliceOptions_PreexistingFieldErrorDoesNotForceNil(t *testing.T) {
	var c vow.Collector
	c.Add(fieldTags, errors.New("recorded by the caller earlier"))

	out := vow.CollectSlice(&c, fieldTags, []string{"a", "b"}, parseNonBlank, vow.NoDuplicates)
	if out == nil {
		t.Fatal("expected the parsed slice back; no option rejected")
	}
}

func TestCollectError_RecoveredByErrorsAs(t *testing.T) {
	var c vow.Collector
	c.Add(fieldInviter, vow.NotBlank(""))
	c.Add(fieldRole, vow.NotBlank(""))

	var ce vow.CollectError
	if !errors.As(c.Err(), &ce) {
		t.Fatal("expected errors.As to recover a CollectError")
	}
	if len(ce) != 2 || ce[0].Field != fieldInviter || ce[1].Field != fieldRole {
		t.Fatalf("got %v, want both fields in order", ce)
	}
}

// TestCollectError_SurvivesWrapping is what makes it usable as a classifier:
// a use case that adds context with %w must not hide the kind of error.
func TestCollectError_SurvivesWrapping(t *testing.T) {
	var c vow.Collector
	c.Add(fieldInviter, vow.NotBlank(""))

	wrapped := fmt.Errorf("creating invite: %w", c.Err())

	var ce vow.CollectError
	if !errors.As(wrapped, &ce) {
		t.Fatal("expected errors.As to reach through fmt.Errorf")
	}
	if !errors.Is(wrapped, vow.ErrBlank) {
		t.Fatal("expected the sentinel to stay reachable through the wrap")
	}
}

// TestCollectError_NilWhenEmpty guards the nil-interface trap: Err is declared
// to return error, so an empty Collector must yield an untyped nil rather than
// a nil CollectError boxed in a non-nil interface.
func TestCollectError_NilWhenEmpty(t *testing.T) {
	var c vow.Collector
	if err := c.Err(); err != nil {
		t.Fatalf("expected untyped nil, got %#v", err)
	}
}

// TestCollectError_RenderingUnchanged pins the format errors.Join produced
// before CollectError replaced it: one field failure per line.
func TestCollectError_RenderingUnchanged(t *testing.T) {
	var c vow.Collector
	c.Add(fieldInviter, vow.NotBlank(""))
	c.Add(fieldRole, vow.NotBlank(""))

	want := "Inviter: is required\nRole: is required"
	if got := c.Err().Error(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestCollectError_IsACopy: mutating what Err handed back must not reach into
// the Collector, which may still be in use.
func TestCollectError_IsACopy(t *testing.T) {
	var c vow.Collector
	c.Add(fieldInviter, vow.NotBlank(""))

	var ce vow.CollectError
	errors.As(c.Err(), &ce)
	ce[0].Field = "Tampered"

	if vow.FieldErrors(c.Err())[0].Field != fieldInviter {
		t.Fatal("mutating the returned CollectError reached the Collector")
	}
}

// TestCollectError_AsStopsAtFirstMatch documents why FieldErrors still exists.
// Joining two command errors leaves errors.As holding only the first, while
// FieldErrors returns every field from both.
func TestCollectError_AsStopsAtFirstMatch(t *testing.T) {
	var a, b vow.Collector
	a.Add(fieldInviter, vow.NotBlank(""))
	b.Add(fieldRole, vow.NotBlank(""))

	joined := errors.Join(a.Err(), b.Err())

	var ce vow.CollectError
	errors.As(joined, &ce)
	if len(ce) != 1 {
		t.Fatalf("expected errors.As to stop at the first collector, got %d", len(ce))
	}
	if fes := vow.FieldErrors(joined); len(fes) != 2 {
		t.Fatalf("expected FieldErrors to return both, got %d", len(fes))
	}
}
