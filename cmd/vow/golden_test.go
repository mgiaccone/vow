package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update golden files instead of comparing against them")

// TestGolden_ValueObject runs the generator against every case directory
// under internal/testdata/valueobject and compares the result to that
// directory's committed fixture_vow_generated.go. Run with -update to refresh
// the golden files after a deliberate generator change; always inspect the
// diff by eye afterward — see CLAUDE.md.
func TestGolden_ValueObject(t *testing.T) {
	runGoldenCases(t, filepath.Join("..", "..", "internal", "testdata", "valueobject"))
}

// TestGolden_Enum covers //vow:enum discovery: a bare directive, one with
// sanitizers and all four flags, and a const block that exercises Go's
// spec-repetition rule (only the first const in the block states its type).
func TestGolden_Enum(t *testing.T) {
	runGoldenCases(t, filepath.Join("..", "..", "internal", "testdata", "enum"))
}

// TestGolden_Mixed covers a single package declaring both a value object
// and an enum, generated into one output file.
func TestGolden_Mixed(t *testing.T) {
	runGoldenCases(t, filepath.Join("..", "..", "internal", "testdata", "mixed"))
}

func runGoldenCases(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("reading %s: %v", root, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(root, name)
			got := generateForTest(t, dir)

			goldenPath := filepath.Join(dir, "fixture_vow_generated.go")
			if *update {
				if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
					t.Fatalf("writing golden file: %v", err)
				}
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("reading golden file %s (run `go test ./cmd/vow -update` to create it): %v", goldenPath, err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("generated output for %s does not match golden file %s\n--- got ---\n%s\n--- want ---\n%s",
					name, goldenPath, got, want)
			}
		})
	}
}

func generateForTest(t *testing.T, dir string) []byte {
	t.Helper()
	p, err := discoverPackage(dir, "fixture_vow_generated.go", "vow", "vow")
	if err != nil {
		t.Fatalf("discoverPackage(%s): %v", dir, err)
	}
	got, err := render(p)
	if err != nil {
		t.Fatalf("render(%s): %v", dir, err)
	}
	return got
}

// TestDeterminism generates the same input twice and asserts byte-identical
// output — the property the documented CI check `go generate ./... && git
// diff --exit-code` depends on.
func TestDeterminism(t *testing.T) {
	dir := filepath.Join("..", "..", "internal", "testdata", "valueobject", "string_sanitize_allflags")
	out1 := generateForTest(t, dir)
	out2 := generateForTest(t, dir)
	if !bytes.Equal(out1, out2) {
		t.Fatal("generation is not deterministic: two runs on the same input produced different output")
	}
}

// TestBootstrap_PackageDoesNotCompile is load-bearing, not incidental: on
// first adoption the user has usually already written types.NewEmail(...)
// somewhere, so the tree is red until generation succeeds. If discovery
// ever grew a dependency on go/types (which requires a type-checkable
// package), this property would break silently for every first-time
// adopter. The fixture below has two independent reasons to fail go build
// — a call to a constructor that doesn't exist yet, and a wholly undefined
// symbol — and discovery must still succeed.
func TestBootstrap_PackageDoesNotCompile(t *testing.T) {
	dir := t.TempDir()
	src := `package fixture

import "github.com/mgiaccone/vow"

var emailSpec = vow.Spec[string]{
	Rules: []vow.Rule[string]{vow.NotBlank},
}

type Email struct {
	v string ` + bq + `vow:"json"` + bq + `
}

func UseEmail() {
	e := NewEmail("test") // NewEmail doesn't exist until generation runs
	_ = e
	_ = someUndefinedSymbol // never defined anywhere; a genuine compile error
}
`
	if err := os.WriteFile(filepath.Join(dir, "types.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := discoverPackage(dir, "fixture_vow_generated.go", "vow", "vow")
	if err != nil {
		t.Fatalf("discovery must succeed on a non-compiling package, got: %v", err)
	}
	got, err := render(p)
	if err != nil {
		t.Fatalf("render must succeed on a non-compiling package, got: %v", err)
	}
	if !strings.Contains(string(got), "func NewEmail(") {
		t.Fatalf("expected generated output to define NewEmail, got:\n%s", got)
	}
}
