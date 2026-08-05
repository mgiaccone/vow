package example_test

import (
	"fmt"
	"testing"

	"github.com/mgiaccone/vow"
	"github.com/mgiaccone/vow/example"
)

func ExampleNewCreateInvite() {
	cmd, err := example.NewCreateInvite("owner@example.com", "new-member@example.com", "admin")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(cmd.Inviter, cmd.Invitee, cmd.Role)
	// Output: owner@example.com new-member@example.com admin
}

func ExampleNewCreateInvite_aggregatedErrors() {
	_, err := example.NewCreateInvite("", "not-an-email", "wizard")
	for _, fe := range vow.FieldErrors(err) {
		fmt.Println(fe)
	}
	// Output:
	// Inviter: is required
	// Invitee: must be a valid email address
	// Role: must be one of: owner, admin, member
}

func TestNewCreateInvite_Success(t *testing.T) {
	cmd, err := example.NewCreateInvite("owner@example.com", "member@example.com", "member")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Inviter.String() != "owner@example.com" {
		t.Errorf("Inviter = %v", cmd.Inviter)
	}
	if cmd.Invitee.String() != "member@example.com" {
		t.Errorf("Invitee = %v", cmd.Invitee)
	}
	if cmd.Role.String() != "member" {
		t.Errorf("Role = %v", cmd.Role)
	}
}

// TestNewCreateInvite_AggregatesThreeFailures is the Collector behavior the
// spec calls out explicitly: three independent field failures collapse
// into one error, and every one of them is recoverable via FieldErrors.
func TestNewCreateInvite_AggregatesThreeFailures(t *testing.T) {
	_, err := example.NewCreateInvite("", "not-an-email", "wizard")
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
	for _, f := range []vow.Field{example.FieldInviter, example.FieldInvitee, example.FieldRole} {
		if !got[f] {
			t.Errorf("expected a FieldError for %s, got none", f)
		}
	}
}

// TestNewCreateInvite_CrossFieldRule is the case a per-type generator
// structurally cannot cover: New<T> constructors validate one value each,
// so "invitee must differ from inviter" can only live in the command
// constructor, checked after both fields have parsed.
func TestNewCreateInvite_CrossFieldRule(t *testing.T) {
	_, err := example.NewCreateInvite("same@example.com", "same@example.com", "admin")
	if err == nil {
		t.Fatal("expected an error when inviter and invitee are the same address")
	}
	fes := vow.FieldErrors(err)
	if len(fes) != 1 || fes[0].Field != example.FieldInvitee {
		t.Fatalf("expected exactly one FieldError on Invitee, got %v", fes)
	}
}

// TestNewCreateInvite_CrossFieldRule_GuardsOnZero confirms the cross-field
// check does not fire spuriously when both fields already failed to parse
// on their own — two zero Emails would otherwise compare equal and add a
// misleading fourth error on top of the two real ones.
func TestNewCreateInvite_CrossFieldRule_GuardsOnZero(t *testing.T) {
	_, err := example.NewCreateInvite("", "", "admin")
	fes := vow.FieldErrors(err)
	if len(fes) != 2 {
		t.Fatalf("expected exactly 2 field errors (both blank, no spurious same-address error), got %d: %v", len(fes), fes)
	}
}
