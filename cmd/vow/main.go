// Command vow generates value objects and enums for Go from a directory of
// hand-written source: struct tags mark value objects, //vow:enum
// directives mark enums. See the module README for the full reference.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("vow", flag.ContinueOnError)
	dir := fs.String("dir", ".", "directory to scan for vow-tagged types and //vow:enum directives")
	out := fs.String("out", "", "output file name, written inside -dir (default: <package>_vow_generated.go)")
	vowQualifier := fs.String("vow-qualifier", "vow", "local qualifier for the vow runtime import in generated code")
	tagKey := fs.String("tag-key", "vow", "struct tag key that marks a value object field")
	verbose := fs.Bool("v", false, "print a one-line summary on success")
	if err := fs.Parse(args); err != nil {
		return 1 // flag has already printed its own usage/error to stderr
	}

	resolvedOut := *out
	if resolvedOut == "" {
		derived, err := derivePackageOutName(*dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "vow: %s\n", err)
			return 1
		}
		resolvedOut = derived
	}

	if err := generate(*dir, resolvedOut, *tagKey, *vowQualifier, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "vow: %s\n", err)
		return 1
	}
	return 0
}

func generate(dir, out, tagKey, vowQualifier string, verbose bool) error {
	p, err := discoverPackage(dir, out, tagKey, vowQualifier)
	if err != nil {
		return err
	}

	src, err := render(p)
	if err != nil {
		return err
	}

	outPath := filepath.Join(dir, filepath.Base(out))

	// Skip the write entirely when nothing changed: this keeps mtimes,
	// file watchers, and build caches quiet, so `go generate ./...` stays
	// cheap enough to run habitually.
	if existing, readErr := os.ReadFile(outPath); readErr == nil && bytes.Equal(existing, src) {
		if verbose {
			fmt.Fprintln(os.Stdout, summaryLine(outPath, p))
		}
		return nil
	}

	if err := os.WriteFile(outPath, src, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}

	if verbose {
		fmt.Fprintln(os.Stdout, summaryLine(outPath, p))
	}
	return nil
}

func summaryLine(outPath string, p *pkg) string {
	return fmt.Sprintf("vow: %s: %d value object%s, %d enum%s",
		outPath, len(p.ValueObjects), plural(len(p.ValueObjects)), len(p.Enums), plural(len(p.Enums)))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
