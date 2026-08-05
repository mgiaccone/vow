// Package example shows vow's runtime and generated code working
// together: value objects and an enum from the types subpackage, parsed
// individually, then combined by a command constructor that reports every
// failure at once and adds the one rule a per-type generator can't
// express on its own.
package example

import (
	"errors"

	"github.com/mgiaccone/vow"
	"github.com/mgiaccone/vow/example/types"
)

// Field constants for CreateInvite. Declaring these beside the struct they
// describe turns a typo into a compile error instead of a mislabeled
// FieldError.
const (
	FieldInviter vow.Field = "Inviter"
	FieldInvitee vow.Field = "Invitee"
	FieldRole    vow.Field = "Role"
)

var errSameAsInviter = errors.New("must not be the same as the inviter")

// CreateInvite is a command to invite a member to a workspace.
type CreateInvite struct {
	Inviter types.Email
	Invitee types.Email
	Role    types.Role
}

// NewCreateInvite parses inviter, invitee, and role independently — each
// failure recorded against its own field rather than stopping at the
// first — and then checks the rule none of those per-field parses can
// express on their own: the invitee must not be the inviter. That
// cross-field check is exactly what vow.Spec and the generator can't do,
// and exactly what vow.Collector is for.
func NewCreateInvite(inviter, invitee, role string) (CreateInvite, error) {
	var c vow.Collector
	cmd := CreateInvite{
		Inviter: vow.Collect(&c, FieldInviter, inviter, types.NewEmail),
		Invitee: vow.Collect(&c, FieldInvitee, invitee, types.NewEmail),
		Role:    vow.Collect(&c, FieldRole, role, types.NewRole),
	}

	// Both fields must have parsed successfully before comparing them:
	// two failed Collects both return the zero Email, so comparing without
	// this guard would report a spurious "same as inviter" error on top
	// of the real per-field failures already recorded.
	if !cmd.Inviter.IsZero() && !cmd.Invitee.IsZero() && cmd.Inviter == cmd.Invitee {
		c.Add(FieldInvitee, errSameAsInviter)
	}

	return cmd, c.Err()
}
