package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update golden files instead of comparing against them")

// TestGolden_ValueObject runs the generator against every case directory
// under internal/testdata/valueobject and compares the result to that
// directory's committed fixture_vow_generated.go. Run with -update to refresh
// the golden files after a deliberate generator change; always inspect the
// diff by eye afterward — see CONTRIBUTING.md.
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

// TestFixturesCompile builds every fixture's hand-written source together
// with the output generated from it. Comparing golden text proves the
// generator is stable; it says nothing about whether what it wrote is valid
// Go, and testdata is invisible to `go build` by design — which is exactly
// how a package of plain value objects came to be emitted with an unused
// runtime import that no test noticed.
//
// The fixtures are assembled into one throwaway module, each case a package,
// so this is a single `go build` rather than one per case. A replace
// directive points at the repo, so nothing is downloaded.
func TestFixturesCompile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping compile check in -short mode")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go toolchain on PATH")
	}

	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolving repo root: %v", err)
	}

	mod := t.TempDir()
	gomod := fmt.Sprintf("module vowfixtures\n\ngo 1.24\n\nrequire github.com/mgiaccone/vow v0.0.0\n\nreplace github.com/mgiaccone/vow => %s\n", repo)
	if err := os.WriteFile(filepath.Join(mod, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	roots := []string{"valueobject", "enum", "mixed"}
	copied := 0
	for _, root := range roots {
		src := filepath.Join(repo, "internal", "testdata", root)
		entries, err := os.ReadDir(src)
		if err != nil {
			t.Fatalf("reading %s: %v", src, err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			// Flattened as <root>_<case> so two roots may share a case name.
			dst := filepath.Join(mod, root+"_"+e.Name())
			if err := os.MkdirAll(dst, 0o755); err != nil {
				t.Fatalf("creating %s: %v", dst, err)
			}
			files, err := filepath.Glob(filepath.Join(src, e.Name(), "*.go"))
			if err != nil {
				t.Fatalf("globbing %s: %v", e.Name(), err)
			}
			for _, f := range files {
				b, err := os.ReadFile(f)
				if err != nil {
					t.Fatalf("reading %s: %v", f, err)
				}
				if err := os.WriteFile(filepath.Join(dst, filepath.Base(f)), b, 0o644); err != nil {
					t.Fatalf("writing %s: %v", f, err)
				}
			}
			copied++
		}
	}
	if copied == 0 {
		t.Fatal("no fixtures found to compile")
	}

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = mod
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated fixtures do not compile (%d packages):\n%s", copied, out)
	}
}
