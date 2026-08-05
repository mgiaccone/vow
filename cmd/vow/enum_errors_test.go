package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRejections_Enum covers every //vow:enum-mode rejection the generator
// makes, asserting on message text.
func TestRejections_Enum(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "directive on a non-string type",
			src: `package fixture

//vow:enum
type Count int

const (
	CountOne Count = 1
)
`,
			want: `//vow:enum is only valid on a string-based type, this type's underlying type is int`,
		},
		{
			name: "no matching const declarations",
			src: `package fixture

//vow:enum
type Role string
`,
			want: `no const declarations of type Role found; //vow:enum requires at least one member`,
		},
		{
			name: "spec= is rejected on an enum",
			src: `package fixture

//vow:enum spec=roleSpec
type Role string

const (
	RoleOwner Role = "owner"
)
`,
			want: `spec= is not valid on //vow:enum; enum membership is generated from the const block`,
		},
		{
			name: "malformed directive",
			src: `package fixture

//vow:enu
type Role string

const (
	RoleOwner Role = "owner"
)
`,
			want: `malformed vow directive "//vow:enu"; expected //vow:enum`,
		},
		{
			name: "json takes a value on an enum",
			src: `package fixture

//vow:enum json=true
type Role string

const (
	RoleOwner Role = "owner"
)
`,
			want: `json does not take a value in this version`,
		},
		{
			name: "unknown sanitizer on an enum",
			src: `package fixture

//vow:enum sanitize=frobnicate
type Role string

const (
	RoleOwner Role = "owner"
)
`,
			want: `unknown sanitizer "frobnicate"`,
		},
		{
			name: "const value is not a string literal",
			src: `package fixture

//vow:enum
type Role string

const prefix = "owner"

const (
	RoleOwner Role = Role(prefix)
)
`,
			want: `must be assigned a string literal to be an enum member`,
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

// TestDetachedEnumDirective is the gotcha spec section 7 calls the first
// thing every user hits: a blank line between the directive and the type
// it should decorate silently detaches it in Go's doc-comment association
// rule. vow catches this explicitly rather than silently generating
// nothing.
func TestDetachedEnumDirective(t *testing.T) {
	src := `package fixture

//vow:enum

type Role string

const (
	RoleOwner Role = "owner"
)
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "types.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := discoverPackage(dir, "fixture_vow_generated.go", "vow", "vow")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "is not attached to any type declaration") {
		t.Fatalf("error = %q", err.Error())
	}
	if !strings.Contains(err.Error(), "blank line") {
		t.Fatalf("error must explain the blank-line gotcha, got %q", err.Error())
	}
}

// TestEnumDirective_AttachedWithNoBlankLine is the correct form paired with
// TestDetachedEnumDirective, so the two tests together show the contrast
// the README's gotchas section documents.
func TestEnumDirective_AttachedWithNoBlankLine(t *testing.T) {
	src := `package fixture

//vow:enum
type Role string

const (
	RoleOwner Role = "owner"
)
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "types.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := discoverPackage(dir, "fixture_vow_generated.go", "vow", "vow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Enums) != 1 || p.Enums[0].Name != "Role" {
		t.Fatalf("expected Role to be discovered as an enum, got %+v", p.Enums)
	}
}
