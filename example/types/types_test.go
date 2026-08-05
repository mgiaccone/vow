package types_test

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/mgiaccone/vow/example/types"
)

// Generated sql-flagged types satisfy database/sql/driver.Valuer and
// database/sql.Scanner with no database-specific code in vow itself. pgx's
// pgtype falls back to these same standard interfaces for types it doesn't
// recognize, so sql-generated types work with native pgx as they are —
// which is why pgx support is this assertion rather than a feature.
//
// This deliberately does not import pgx: proving compatibility that way
// would put a third-party dependency in the module, and satisfying the
// interfaces is the whole contract.
var (
	_ driver.Valuer = types.Email{}
	_ sql.Scanner   = (*types.Email)(nil)
	_ driver.Valuer = types.Quantity{}
	_ sql.Scanner   = (*types.Quantity)(nil)
	_ driver.Valuer = types.Role("")
	_ sql.Scanner   = (*types.Role)(nil)
)

func ExampleNewEmail() {
	e, err := types.NewEmail("  USER@Example.com  ")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(e)
	// Output: user@example.com
}

func ExampleNewRole() {
	r, err := types.NewRole("  ADMIN  ")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(r)
	// Output: admin
}

func TestEmail_SanitizeThenValidate(t *testing.T) {
	e, err := types.NewEmail("  USER@Example.com  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Unwrap() != "user@example.com" {
		t.Fatalf("got %q, want %q", e.Unwrap(), "user@example.com")
	}
}

func TestEmail_Invalid(t *testing.T) {
	if _, err := types.NewEmail("not-an-email"); err == nil {
		t.Fatal("expected an error for a malformed address")
	}
}

func TestEmail_JSON_PlainScalar(t *testing.T) {
	e := types.MustEmail("a@b.com")
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"a@b.com"` {
		t.Fatalf("got %s, want a plain JSON string scalar", b)
	}
}

func TestEmail_SQLRoundTrip(t *testing.T) {
	e := types.MustEmail("a@b.com")
	v, err := e.Value()
	if err != nil {
		t.Fatal(err)
	}
	var e2 types.Email
	if err := e2.Scan(v); err != nil {
		t.Fatal(err)
	}
	if e2 != e {
		t.Fatalf("round trip mismatch: %v != %v", e2, e)
	}
}

func TestEmail_IsZero_FailedParseAndFreshZero(t *testing.T) {
	var fresh types.Email
	if !fresh.IsZero() {
		t.Fatal("a fresh zero value must report IsZero")
	}

	failed, err := types.NewEmail("")
	if err == nil {
		t.Fatal("expected an error for a blank address")
	}
	if !failed.IsZero() {
		t.Fatal("a failed parse must return the zero value")
	}
}

func TestQuantity_SQLRoundTrip(t *testing.T) {
	q := types.MustQuantity(5)
	v, err := q.Value()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := v.(int64); !ok {
		t.Fatalf("expected int64 driver.Value, got %T", v)
	}
	var q2 types.Quantity
	if err := q2.Scan(v); err != nil {
		t.Fatal(err)
	}
	if q2 != q {
		t.Fatalf("round trip mismatch: %v != %v", q2, q)
	}
}

func TestQuantity_MustBePositive(t *testing.T) {
	if _, err := types.NewQuantity(0); err == nil {
		t.Fatal("expected an error for a non-positive quantity")
	}
}

func TestExpiry_MustBeFuture(t *testing.T) {
	if _, err := types.NewExpiry(time.Now().Add(-time.Hour)); err == nil {
		t.Fatal("expected an error for a past expiry")
	}
	if _, err := types.NewExpiry(time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("unexpected error for a future expiry: %v", err)
	}
}

func TestRole_SanitizeThenValidate(t *testing.T) {
	r, err := types.NewRole("  ADMIN  ")
	if err != nil {
		t.Fatal(err)
	}
	if r != types.RoleAdmin {
		t.Fatalf("got %v, want %v", r, types.RoleAdmin)
	}
}

func TestRole_JSON_PlainScalar(t *testing.T) {
	b, err := json.Marshal(types.RoleOwner)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"owner"` {
		t.Fatalf("got %s, want a plain JSON string scalar", b)
	}
}

// TestRole_ScanRetiredMember is the SQL-side counterpart to
// TestEmail_IsZero_FailedParseAndFreshZero: Scan never validates, because
// retiring an enum member is routine and must not make old rows
// unreadable — including the rows you'd need to load to fix them.
func TestRole_ScanRetiredMember(t *testing.T) {
	var r types.Role
	if err := r.Scan("founder"); err != nil {
		t.Fatalf("Scan must accept a retired/unknown member, got error: %v", err)
	}
	if r.IsValid() {
		t.Fatal("a retired member must report IsValid() == false")
	}
	if r != types.Role("founder") {
		t.Fatalf("Scan must not alter or reject the value, got %v", r)
	}
}

func TestRole_IsZero(t *testing.T) {
	var r types.Role
	if !r.IsZero() {
		t.Fatal("zero value must report IsZero")
	}
	if r.IsValid() {
		t.Fatal("the zero value must never be valid")
	}
}

func TestRoleValues(t *testing.T) {
	got := types.RoleValues()
	want := []types.Role{types.RoleOwner, types.RoleAdmin, types.RoleMember}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
