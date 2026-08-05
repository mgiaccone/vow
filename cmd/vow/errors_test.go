package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const bq = "`"

// TestRejections_ValueObject covers every value-object-mode rejection the
// generator makes, asserting on message text so a regression shows up as a
// diff in an expected sentence, not just "test failed."
func TestRejections_ValueObject(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "exported field",
			src: `package fixture
type Email struct {
	V string ` + bq + `vow:"json"` + bq + `
}
`,
			want: `field "V" must be unexported, otherwise the constructor can be bypassed`,
		},
		{
			name: "two fields tagged",
			src: `package fixture
type Email struct {
	a string ` + bq + `vow:"json"` + bq + `
	b string ` + bq + `vow:"json"` + bq + `
}
`,
			want: `struct has 2 fields tagged "vow"; a value object must have exactly one`,
		},
		{
			name: "multi-field struct, one tagged",
			src: `package fixture
type Pair struct {
	a string ` + bq + `vow:"json"` + bq + `
	b string
}
`,
			want: `value object struct must have exactly one field, has 2`,
		},
		{
			name: "field with two names",
			src: `package fixture
type Pair struct {
	a, b string ` + bq + `vow:"json"` + bq + `
}
`,
			want: `must declare exactly one name`,
		},
		{
			name: "non-comparable base: slice",
			src: `package fixture
type Tags struct {
	v []string ` + bq + `vow:"json"` + bq + `
}
`,
			want: `base type []string is not comparable`,
		},
		{
			name: "non-comparable base: map",
			src: `package fixture
type Attrs struct {
	v map[string]string ` + bq + `vow:"json"` + bq + `
}
`,
			want: `base type map[string]string is not comparable`,
		},
		{
			name: "sanitize on non-string base",
			src: `package fixture
import "github.com/mgiaccone/vow"
var countSpec = vow.Spec[int]{}
type Count struct {
	v int ` + bq + `vow:"sanitize=trim"` + bq + `
}
`,
			want: `sanitize= is only valid on a string base, this type's base is int`,
		},
		{
			name: "unknown sanitizer",
			src: `package fixture
import "github.com/mgiaccone/vow"
var nameSpec = vow.Spec[string]{}
type Name struct {
	v string ` + bq + `vow:"sanitize=frobnicate"` + bq + `
}
`,
			want: `unknown sanitizer "frobnicate"; known sanitizers are trim, lower, upper, collapse`,
		},
		{
			name: "derived spec var missing",
			src: `package fixture
type Email struct {
	v string ` + bq + `vow:"json"` + bq + `
}
`,
			want: `no var named emailSpec found`,
		},
		{
			name: "explicit spec= typo",
			src: `package fixture
import "github.com/mgiaccone/vow"
var realSpec = vow.Spec[string]{}
type Email struct {
	v string ` + bq + `vow:"spec=typoSpec"` + bq + `
}
`,
			want: `spec=typoSpec names a var that does not exist in this package`,
		},
		{
			name: "json takes a value",
			src: `package fixture
type Email struct {
	v string ` + bq + `vow:"json=true"` + bq + `
}
`,
			want: `json does not take a value in this version`,
		},
		{
			name: "sql takes a value",
			src: `package fixture
type Email struct {
	v string ` + bq + `vow:"sql=pgx"` + bq + `
}
`,
			want: `sql does not take a value in this version`,
		},
		{
			name: "text takes a value",
			src: `package fixture
type Email struct {
	v string ` + bq + `vow:"text=rfc3339"` + bq + `
}
`,
			want: `text does not take a value in this version`,
		},
		{
			name: "unknown tag option",
			src: `package fixture
type Email struct {
	v string ` + bq + `vow:"frobnicate"` + bq + `
}
`,
			want: `unknown tag option "frobnicate"`,
		},
		{
			name: "uint64 base with sql",
			src: `package fixture
import "github.com/mgiaccone/vow"
var idSpec = vow.Spec[uint64]{}
type ID struct {
	v uint64 ` + bq + `vow:"sql"` + bq + `
}
`,
			want: `sql is not supported on a uint64 base`,
		},
		{
			name: "uint base with sql",
			src: `package fixture
import "github.com/mgiaccone/vow"
var idSpec = vow.Spec[uint]{}
type ID struct {
	v uint ` + bq + `vow:"sql"` + bq + `
}
`,
			want: `sql is not supported on a uint base`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "types.go"), []byte(c.src), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := discoverPackage(dir, "fixture_vow_generated.go", "vow", "vow")
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error = %q\nwant substring %q", err.Error(), c.want)
			}
		})
	}
}

// TestRejections_NameCollision covers identifiers generation would declare
// that the hand-written package already uses. Go catches these anyway, but
// as a redeclaration error inside the generated file — code the user didn't
// write; vow reports them against the declaration they can change.
func TestRejections_NameCollision(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "parser var for a sanitized value object",
			src: `package fixture
import "github.com/mgiaccone/vow"
var emailSpec = vow.Spec[string]{}
var emailParser = "mine"
type Email struct {
	v string ` + bq + `vow:"sanitize=trim"` + bq + `
}
`,
			want: `generated code declares emailParser, but that name is already declared at`,
		},
		{
			name: "drop-sanitize remedy is offered for a parser collision",
			src: `package fixture
import "github.com/mgiaccone/vow"
var emailSpec = vow.Spec[string]{}
var emailParser = "mine"
type Email struct {
	v string ` + bq + `vow:"sanitize=trim"` + bq + `
}
`,
			want: `drop sanitize= from the tag`,
		},
		{
			name: "handwritten constructor",
			src: `package fixture
import "github.com/mgiaccone/vow"
var emailSpec = vow.Spec[string]{}
func NewEmail(s string) (Email, error) { return Email{}, nil }
type Email struct {
	v string ` + bq + `vow:"json"` + bq + `
}
`,
			want: `generated code declares NewEmail, but that name is already declared at`,
		},
		{
			name: "handwritten Must constructor",
			src: `package fixture
import "github.com/mgiaccone/vow"
var emailSpec = vow.Spec[string]{}
func MustEmail(s string) Email { return Email{} }
type Email struct {
	v string ` + bq + `vow:"json"` + bq + `
}
`,
			want: `generated code declares MustEmail`,
		},
		{
			name: "enum Values function",
			src: `package fixture

//vow:enum
type Role string

const (
	RoleOwner Role = "owner"
)

func RoleValues() []Role { return nil }
`,
			want: `generated code declares RoleValues`,
		},
		{
			name: "enum values slice",
			src: `package fixture

//vow:enum
type Role string

const (
	RoleOwner  Role = "owner"
	roleValues Role = "collides"
)
`,
			want: `generated code declares roleValues`,
		},
		{
			name: "enum parser var",
			src: `package fixture

//vow:enum
type Role string

const (
	RoleOwner Role = "owner"
)

var roleParser = "mine"
`,
			want: `generated code declares roleParser`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "types.go"), []byte(c.src), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := discoverPackage(dir, "fixture_vow_generated.go", "vow", "vow")
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error = %q\nwant substring %q", err.Error(), c.want)
			}
		})
	}
}

// TestNoCollision_MethodsAndUnsanitizedTypes guards the check against false
// positives: a method named like a generated helper lives in its receiver's
// namespace, not the package's, and a value object without sanitize=
// generates no parser var at all, so an unrelated emailParser is fine.
func TestNoCollision_MethodsAndUnsanitizedTypes(t *testing.T) {
	src := `package fixture

import "github.com/mgiaccone/vow"

var emailSpec = vow.Spec[string]{}

type Email struct {
	v string ` + bq + `vow:"json"` + bq + `
}

type other struct{}

// A method, not a package-level identifier: must not collide.
func (other) NewEmail() {}

// No sanitize= on Email, so no emailParser is generated.
var emailParser = "unrelated"
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "types.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := discoverPackage(dir, "fixture_vow_generated.go", "vow", "vow"); err != nil {
		t.Fatalf("expected no collision, got: %v", err)
	}
}

func TestNoTypesFound(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "types.go"), []byte("package fixture\n\nfunc Foo() int { return 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := discoverPackage(dir, "fixture_vow_generated.go", "vow", "vow")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "no vow-tagged types or //vow:enum directives found") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestNoTypesFound_StaleOutputNamed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "types.go"), []byte("package fixture\n\nfunc Foo() int { return 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stalePath := filepath.Join(dir, "fixture_vow_generated.go")
	if err := os.WriteFile(stalePath, []byte("// Code generated by vow. DO NOT EDIT.\npackage fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := discoverPackage(dir, "fixture_vow_generated.go", "vow", "vow")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), stalePath) {
		t.Fatalf("error must name the stale file path %q, got %q", stalePath, err.Error())
	}
	if !strings.Contains(err.Error(), "delete") {
		t.Fatalf("error must say the stale file can be deleted, got %q", err.Error())
	}
}
