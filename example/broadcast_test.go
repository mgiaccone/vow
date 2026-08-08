package example_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/mgiaccone/vow"
	"github.com/mgiaccone/vow/example"
)

func ExampleNewSendBroadcast() {
	cmd, err := example.NewSendBroadcast("boss@example.com", []string{"a@example.com", "b@example.com"})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(cmd.Sender, len(cmd.Recipients))
	// Output: boss@example.com 2
}

// ExampleNewSendBroadcast_badElements shows what a consumer that knows
// nothing about ElementErrors sees: one entry per field, with every bad
// element rendered into it.
func ExampleNewSendBroadcast_badElements() {
	_, err := example.NewSendBroadcast("boss@example.com", []string{"a@example.com", "nope", "c@example.com", "also bad"})
	for _, fe := range vow.FieldErrors(err) {
		fmt.Printf("%s: %s\n", fe.Field, fe.Err)
	}
	// Output: Recipients: [1] must be a valid email address; [3] must be a valid email address
}

// TestSendBroadcast_TrueIndices is the property CollectSlice exists to hold:
// the reported positions are the elements the caller sent, not the ordinal of
// each failure.
func TestSendBroadcast_TrueIndices(t *testing.T) {
	_, err := example.NewSendBroadcast(
		"boss@example.com",
		[]string{"a@example.com", "nope", "c@example.com", "also bad"},
	)

	fes := vow.FieldErrors(err)
	if len(fes) != 1 {
		t.Fatalf("expected one entry for the field, got %d: %v", len(fes), fes)
	}

	var ee vow.ElementErrors
	if !errors.As(fes[0].Err, &ee) {
		t.Fatal("expected the field error to carry an ElementErrors")
	}
	if len(ee) != 2 || ee[0].Index != 1 || ee[1].Index != 3 {
		t.Fatalf("got indices %v, want elements 1 and 3", ee)
	}
}

// TestSendBroadcast_NilRatherThanShortened: a caller who ignores the error
// must not end up with a partial audience.
func TestSendBroadcast_NilRatherThanShortened(t *testing.T) {
	cmd, err := example.NewSendBroadcast("boss@example.com", []string{"a@example.com", "nope"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if cmd.Recipients != nil {
		t.Fatalf("expected nil recipients on failure, got %v", cmd.Recipients)
	}
}

func TestSendBroadcast_SenderAndRecipientsBothReported(t *testing.T) {
	_, err := example.NewSendBroadcast("not-an-email", []string{"nope"})

	fes := vow.FieldErrors(err)
	if len(fes) != 2 {
		t.Fatalf("expected both fields reported, got %d: %v", len(fes), fes)
	}
	if !errors.Is(err, vow.ErrNotMatch) {
		t.Fatal("expected the rule sentinel to stay reachable through ElementErrors")
	}
}

// ExampleNewSendBroadcast_duplicates shows the check catching a repeat that
// only became one after sanitizing: "  Boss@Example.com " and
// "boss@example.com" are different strings but the same Email.
func ExampleNewSendBroadcast_duplicates() {
	_, err := example.NewSendBroadcast("sender@example.com", []string{
		"boss@example.com",
		"team@example.com",
		"  Boss@Example.com ",
	})
	for _, fe := range vow.FieldErrors(err) {
		fmt.Printf("%s: %s\n", fe.Field, fe.Err)
	}
	// Output: Recipients: [2] duplicates item 0
}

func TestSendBroadcast_DuplicateIsReportedNotRemoved(t *testing.T) {
	cmd, err := example.NewSendBroadcast("sender@example.com", []string{
		"a@example.com", "a@example.com",
	})
	if err == nil {
		t.Fatal("expected a duplicate to be reported")
	}
	if cmd.Recipients != nil {
		t.Fatalf("expected nil rather than a silently shortened list, got %v", cmd.Recipients)
	}
	if !errors.Is(err, vow.ErrDuplicate) {
		t.Fatal("expected errors.Is to reach ErrDuplicate")
	}
}
