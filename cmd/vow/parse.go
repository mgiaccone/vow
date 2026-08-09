package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// discoverPackage parses every eligible .go file in dir and resolves the
// value objects and enums declared there. It never uses go/types to
// type-check: types.ExprString is the only use of the go/types package,
// and it is a pure syntactic printer, not a checker. That is deliberate —
// see CONTRIBUTING.md — because the generator must work on a package that
// does not compile.
func discoverPackage(dir, outFile, tagKey, vowQualifier string) (*pkg, error) {
	fset, files, pkgName, err := loadPackage(dir, outFile)
	if err != nil {
		return nil, err
	}

	decls := collectPackageDecls(fset, files)
	imports := newImportCollector()

	var valueObjects []*valueObject
	enumCandidates := map[string]*enumType{}
	usedDocs := map[*ast.CommentGroup]bool{}

	for _, f := range files {
		table := importTable(f)
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			singleSpec := len(gd.Specs) == 1
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				doc := ts.Doc
				if doc == nil && singleSpec {
					doc = gd.Doc
				}
				if doc != nil {
					usedDocs[doc] = true
				}

				if st, ok := ts.Type.(*ast.StructType); ok {
					vo, err := resolveValueObject(fset, table, ts, st, tagKey, decls, imports)
					if err != nil {
						return nil, err
					}
					if vo != nil {
						valueObjects = append(valueObjects, vo)
					}
					continue
				}

				enum, err := resolveEnumCandidate(fset, ts, doc)
				if err != nil {
					return nil, err
				}
				if enum == nil {
					continue
				}
				if _, dup := enumCandidates[enum.Name]; dup {
					return nil, errAt(enum.Pos, enum.Name, "duplicate //vow:enum directive for this type")
				}
				enumCandidates[enum.Name] = enum
			}
		}
	}

	for _, f := range files {
		if err := detectDetachedEnumDirective(fset, f, usedDocs); err != nil {
			return nil, err
		}
	}

	if err := collectConstMembers(fset, files, enumCandidates); err != nil {
		return nil, err
	}
	for _, e := range enumCandidates {
		if len(e.Members) == 0 {
			return nil, errAt(e.Pos, e.Name, "no const declarations of type %s found; //vow:enum requires at least one member", e.Name)
		}
	}

	if len(valueObjects) == 0 && len(enumCandidates) == 0 {
		if _, statErr := os.Stat(filepath.Join(dir, filepath.Base(outFile))); statErr == nil {
			return nil, fmt.Errorf("no vow-tagged types or //vow:enum directives found in %s, but %s already exists from a previous run; vow never deletes a file, so if you removed the last value object or enum on purpose, delete %[2]s yourself",
				dir, filepath.Join(dir, filepath.Base(outFile)))
		}
		return nil, fmt.Errorf("no vow-tagged types or //vow:enum directives found in %s", dir)
	}

	sort.Slice(valueObjects, func(i, j int) bool { return valueObjects[i].Name < valueObjects[j].Name })
	enums := make([]*enumType, 0, len(enumCandidates))
	for _, e := range enumCandidates {
		enums = append(enums, e)
	}
	sort.Slice(enums, func(i, j int) bool { return enums[i].Name < enums[j].Name })

	// After sorting, so which collision is reported first is deterministic.
	if err := checkNameCollisions(decls, valueObjects, enums); err != nil {
		return nil, err
	}

	return &pkg{
		Name:          pkgName,
		VowImportPath: "github.com/mgiaccone/vow",
		VowQualifier:  vowQualifier,
		Imports:       imports.sorted(),
		ValueObjects:  valueObjects,
		Enums:         enums,
	}, nil
}

// loadPackage parses every non-test .go file in dir except outFile (matched
// by base name, so stale output can never influence discovery), sorted by
// filename for deterministic output.
func loadPackage(dir, outFile string) (*token.FileSet, []*ast.File, string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, "", fmt.Errorf("reading %s: %w", dir, err)
	}
	outBase := filepath.Base(outFile)

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == outBase {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, nil, "", fmt.Errorf("no Go source files found in %s", dir)
	}

	fset := token.NewFileSet()
	var files []*ast.File
	var pkgName string
	for _, name := range names {
		path := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil, nil, "", fmt.Errorf("parsing %s: %w", path, err)
		}
		if pkgName == "" {
			pkgName = f.Name.Name
		} else if f.Name.Name != pkgName {
			// A mismatched package name in the same directory is a
			// pre-existing build error the generator isn't responsible
			// for diagnosing; skip the file rather than fail discovery.
			continue
		}
		files = append(files, f)
	}
	return fset, files, pkgName, nil
}

// derivePackageOutName peeks at the package clause of the first (sorted)
// non-test .go file in dir and returns "<package>_vow_generated.go" — the
// default -out name when the flag isn't set explicitly. It doesn't need to
// exclude any previous output file first: if one exists, its package
// clause already matches the real package name, so including it in this
// peek is harmless. parser.PackageClauseOnly stops right after the package
// clause, so this is cheap even though it's a throwaway parse distinct from
// loadPackage's real one.
func derivePackageOutName(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", dir, err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "", fmt.Errorf("no Go source files found in %s", dir)
	}

	path := filepath.Join(dir, names[0])
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly)
	if err != nil {
		return "", fmt.Errorf("parsing %s: %w", path, err)
	}
	return f.Name.Name + "_vow_generated.go", nil
}

// packageDecls indexes the hand-written package's declarations three ways:
// vars and funcs each resolve a spec= or derived spec name — a spec may be
// either — and all holds every package-level identifier, since var, const,
// type, and func share one namespace in Go, so generation can tell when it
// is about to redeclare one.
type packageDecls struct {
	vars  map[string]bool
	funcs map[string]*ast.FuncDecl
	all   map[string]token.Position
}

func collectPackageDecls(fset *token.FileSet, files []*ast.File) packageDecls {
	decls := packageDecls{
		vars:  map[string]bool{},
		funcs: map[string]*ast.FuncDecl{},
		all:   map[string]token.Position{},
	}
	record := func(ident *ast.Ident) {
		if ident.Name == "_" {
			return
		}
		if _, seen := decls.all[ident.Name]; !seen {
			decls.all[ident.Name] = fset.Position(ident.Pos())
		}
	}

	for _, f := range files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv != nil {
					continue // a method lives in its receiver's namespace, not the package's
				}
				if _, seen := decls.funcs[d.Name.Name]; !seen {
					decls.funcs[d.Name.Name] = d
				}
				record(d.Name)
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.ValueSpec:
						for _, name := range s.Names {
							if d.Tok == token.VAR {
								decls.vars[name.Name] = true
							}
							record(name)
						}
					case *ast.TypeSpec:
						record(s.Name)
					}
				}
			}
		}
	}
	return decls
}

// checkNameCollisions rejects a package that already declares an identifier
// generation would declare itself. Go catches these anyway, but as a
// redeclaration error inside the generated file — code the user did not
// write and cannot fix. Reporting it here points at the declaration they
// can actually change, and says which tag option is responsible.
func checkNameCollisions(decls packageDecls, valueObjects []*valueObject, enums []*enumType) error {
	claim := func(pos token.Position, typeName, generated, remedy string) error {
		existing, taken := decls.all[generated]
		if !taken {
			return nil
		}
		return errAt(pos, typeName, "generated code declares %s, but that name is already declared at %s; %s",
			generated, existing, remedy)
	}

	for _, vo := range valueObjects {
		const rename = "rename the existing declaration"
		generated := []string{"New" + vo.Name, "Must" + vo.Name}
		if vo.HasGenerator() {
			generated = append(generated, "Generate"+vo.Name)
		}
		for _, name := range generated {
			if err := claim(vo.Pos, vo.Name, name, rename); err != nil {
				return err
			}
		}
		if parser := vo.ParserVar(); parser != "" {
			if err := claim(vo.Pos, vo.Name, parser,
				"rename the existing declaration, or drop sanitize= from the tag to stop generating a parser"); err != nil {
				return err
			}
		}
		if sanitizer := vo.SanitizerVar(); sanitizer != "" {
			if err := claim(vo.Pos, vo.Name, sanitizer,
				"rename the existing declaration, or drop sanitize= from the tag to stop generating a sanitizer"); err != nil {
				return err
			}
		}
	}

	for _, e := range enums {
		const rename = "rename the existing declaration"
		generated := []string{"New" + e.Name, "Must" + e.Name, e.Name + "Values", e.ParserVar(), e.ValuesVar()}
		for _, name := range generated {
			if err := claim(e.Pos, e.Name, name, rename); err != nil {
				return err
			}
		}
	}
	return nil
}

// importTable maps a file's local package identifiers to their import
// paths, e.g. "time" -> "time", or "t" -> "time" for `import t "time"`.
// Unaliased imports are keyed by the last path segment, which is a
// heuristic — the generator has no type info to learn a package's real
// clause name from — but it is correct for the overwhelming majority of
// packages, including the standard library.
func importTable(f *ast.File) map[string]string {
	table := map[string]string{}
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		local := defaultLocalName(path)
		if imp.Name != nil {
			local = imp.Name.Name
		}
		if local == "_" || local == "." {
			continue
		}
		table[local] = path
	}
	return table
}

func defaultLocalName(path string) string {
	if idx := strings.LastIndexByte(path, '/'); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

// resolveValueObject inspects a StructType TypeSpec and returns a
// *valueObject if exactly one field carries tagKey, nil if none do (this
// struct simply isn't a vow type), or an error for anything in between.
func resolveValueObject(fset *token.FileSet, table map[string]string, ts *ast.TypeSpec, st *ast.StructType, tagKey string, decls packageDecls, imports *importCollector) (*valueObject, error) {
	typeName := ts.Name.Name
	pos := fset.Position(ts.Pos())

	var taggedField *ast.Field
	taggedCount := 0
	for _, field := range st.Fields.List {
		if field.Tag == nil {
			continue
		}
		raw, err := strconv.Unquote(field.Tag.Value)
		if err != nil {
			continue
		}
		if _, ok := reflect.StructTag(raw).Lookup(tagKey); ok {
			taggedField = field
			taggedCount++
		}
	}
	if taggedCount == 0 {
		return nil, nil
	}
	if taggedCount > 1 {
		return nil, errAt(pos, typeName, "struct has %d fields tagged %q; a value object must have exactly one", taggedCount, tagKey)
	}
	if len(st.Fields.List) != 1 {
		return nil, errAt(pos, typeName, "value object struct must have exactly one field, has %d; multi-field value objects are out of scope for the generator — declare the extra fields on a plain struct and validate them with vow.Collector and a handwritten constructor instead", len(st.Fields.List))
	}

	field := taggedField
	if len(field.Names) != 1 {
		return nil, errAt(pos, typeName, "value object field must declare exactly one name")
	}
	nameIdent := field.Names[0]
	if nameIdent.IsExported() {
		return nil, errAt(pos, typeName, "field %q must be unexported, otherwise the constructor can be bypassed", nameIdent.Name)
	}

	raw, _ := strconv.Unquote(field.Tag.Value)
	tagValue, _ := reflect.StructTag(raw).Lookup(tagKey)
	opts, err := resolveOptions(parseTagOptions(tagValue), true, pos, typeName)
	if err != nil {
		return nil, err
	}

	baseExpr := types.ExprString(field.Type)
	kind := classifyBase(baseExpr)

	if len(opts.Sanitizers) > 0 && baseExpr != "string" {
		return nil, errAt(pos, typeName, "sanitize= is only valid on a string base, this type's base is %s", baseExpr)
	}
	if opts.HasSQL && kind == kindUintWide {
		return nil, errAt(pos, typeName, "sql is not supported on a %s base: it may not fit losslessly in the int64 that database/sql/driver.Value requires; use int64 or a narrower unsigned width instead", baseExpr)
	}
	if !isComparableExpr(field.Type) {
		return nil, errAt(pos, typeName, "base type %s is not comparable; IsZero and == require a comparable base, so slices, maps, funcs, and channels cannot be used", baseExpr)
	}

	specVar := opts.SpecOverride
	derived := !opts.HasSpec
	if derived {
		specVar = specVarName(typeName)
	}

	// A spec may be a package-level var (no parameters) or a func returning
	// one, whose parameters are appended to the generated constructors after
	// the value. The func's return type is not checked — that would need
	// go/types — so a mismatch surfaces as a compile error in the generated
	// file, exactly as a mismatched Spec[int]/string base already does.
	var specParams []specParam
	var specIsFuncDecl bool
	specFn, isFunc := decls.funcs[specVar]
	switch {
	case decls.vars[specVar]:
		// nothing to do: today's path
	case isFunc:
		specIsFuncDecl = true
		specParams, err = specFuncParams(fset, table, specFn, pos, typeName, imports)
		if err != nil {
			return nil, err
		}
	default:
		if _, exists := decls.all[specVar]; exists {
			return nil, errAt(pos, typeName, "%s is declared in this package but is neither a var nor a func; a spec must be var %s = vow.Spec[%s]{...} or func %s(...) vow.Spec[%s]", specVar, specVar, baseExpr, specVar, baseExpr)
		}
		if derived {
			return nil, errAt(pos, typeName, "no var or func named %s found; declare var %s = vow.Spec[%s]{...} in the package — or func %s(...) vow.Spec[%s] to take validation parameters — or set spec=NAME in the tag to use a different name", specVar, specVar, baseExpr, specVar, baseExpr)
		}
		return nil, errAt(pos, typeName, "spec=%s names a var or func that does not exist in this package", specVar)
	}

	// A generator is optional and found by name, so its absence is silent:
	// the type simply gets no Generate<T>. A caller who expected one is
	// calling Generate<T>, so a misspelled generator surfaces as a compile
	// error at the call site rather than as wrong behavior here.
	var generatorFunc string
	genName := generatorFuncName(typeName)
	if genFn, ok := decls.funcs[genName]; ok {
		if genFn.Type.Params != nil && len(genFn.Type.Params.List) > 0 {
			return nil, errAt(fset.Position(genFn.Pos()), typeName, "generator %s must take no parameters; a value that needs one to be built is not one vow can mint, and parameters that select rules belong on the spec func instead", genName)
		}
		generatorFunc = genName
	}

	if selExpr, ok := field.Type.(*ast.SelectorExpr); ok {
		if ident, ok := selExpr.X.(*ast.Ident); ok {
			if path, ok := table[ident.Name]; ok {
				if err := imports.add(ident.Name, path); err != nil {
					return nil, errAt(pos, typeName, "%s", err)
				}
			}
		}
	}

	return &valueObject{
		Name:           typeName,
		Pos:            pos,
		BaseType:       baseExpr,
		BaseKind:       kind,
		FieldName:      nameIdent.Name,
		SpecVar:        specVar,
		SpecParams:     specParams,
		SpecIsFuncDecl: specIsFuncDecl,
		GeneratorFunc:  generatorFunc,
		Sanitizers:     opts.Sanitizers,
		HasJSON:        opts.HasJSON,
		HasSQL:         opts.HasSQL,
		HasText:        opts.HasText,
	}, nil
}

// specFuncParams renders a spec func's parameter list for the generated
// constructors. Everything here is syntactic: names and types come straight
// from the AST, with types printed by types.ExprString exactly as the base
// type is.
//
// Qualified parameter types pull their package into the output file's
// imports, the same way a qualified base type does — without this the
// generated file would name a package it never imported.
func specFuncParams(fset *token.FileSet, table map[string]string, fn *ast.FuncDecl, pos token.Position, typeName string, imports *importCollector) ([]specParam, error) {
	if fn.Type.Params == nil {
		return nil, nil
	}

	var params []specParam
	for _, field := range fn.Type.Params.List {
		typeExpr := types.ExprString(field.Type)

		// A variadic parameter is unambiguous here: the generated value
		// argument comes first, so ...T can only be trailing.
		elem := field.Type
		if ell, ok := elem.(*ast.Ellipsis); ok {
			elem = ell.Elt
		}
		if err := collectExprImports(table, elem, imports, pos, typeName); err != nil {
			return nil, err
		}

		if len(field.Names) == 0 {
			// An unnamed parameter still needs a name in the generated
			// signature; position is stable, so this stays deterministic.
			params = append(params, specParam{Name: fmt.Sprintf("p%d", len(params)), Type: typeExpr})
			continue
		}
		for _, name := range field.Names {
			if name.Name == "in" {
				return nil, errAt(fset.Position(name.Pos()), typeName, "spec func parameter %q shadows the value argument in the generated constructor; rename it", name.Name)
			}
			if name.Name == "_" {
				params = append(params, specParam{Name: fmt.Sprintf("p%d", len(params)), Type: typeExpr})
				continue
			}
			params = append(params, specParam{Name: name.Name, Type: typeExpr})
		}
	}
	return params, nil
}

// collectExprImports registers the package of any qualified name appearing
// in expr, so the generated file imports what it references.
func collectExprImports(table map[string]string, expr ast.Expr, imports *importCollector, pos token.Position, typeName string) error {
	var walkErr error
	ast.Inspect(expr, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		path, ok := table[ident.Name]
		if !ok {
			return true
		}
		if err := imports.add(ident.Name, path); err != nil {
			walkErr = errAt(pos, typeName, "%s", err)
			return false
		}
		return true
	})
	return walkErr
}

// isComparableExpr is a syntactic approximation of comparability: it
// rejects slices, maps, funcs, and channels (and structs/arrays built from
// them), without needing go/types to resolve named types fully.
func isComparableExpr(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.ArrayType:
		if t.Len == nil {
			return false // a slice
		}
		return isComparableExpr(t.Elt)
	case *ast.MapType, *ast.FuncType, *ast.ChanType:
		return false
	case *ast.StructType:
		for _, f := range t.Fields.List {
			if !isComparableExpr(f.Type) {
				return false
			}
		}
		return true
	default:
		return true
	}
}

// resolveEnumCandidate looks for a //vow:enum directive in doc and, if
// found, validates the directive and the type it decorates. It returns nil
// (not an error) when doc carries no vow directive at all.
func resolveEnumCandidate(fset *token.FileSet, ts *ast.TypeSpec, doc *ast.CommentGroup) (*enumType, error) {
	typeName := ts.Name.Name
	pos := fset.Position(ts.Pos())

	rest, ok, malformed, malformedText := findVowEnumDirective(doc)
	if malformed {
		return nil, errAt(pos, typeName, "malformed vow directive %q; expected //vow:enum with optional space-separated options", malformedText)
	}
	if !ok {
		return nil, nil
	}

	baseExpr := types.ExprString(ts.Type)
	if baseExpr != "string" {
		return nil, errAt(pos, typeName, "//vow:enum is only valid on a string-based type, this type's underlying type is %s", baseExpr)
	}

	opts, err := resolveOptions(parseTagOptions(rest), false, pos, typeName)
	if err != nil {
		return nil, err
	}

	return &enumType{
		Name:       typeName,
		Pos:        pos,
		Sanitizers: opts.Sanitizers,
		HasJSON:    opts.HasJSON,
		HasSQL:     opts.HasSQL,
		HasText:    opts.HasText,
	}, nil
}

// findVowEnumDirective scans doc for a line beginning "//vow:". Go
// directives (//go:generate and the like) require no space after the
// slashes to be recognized, and vow follows the same convention: "//
// vow:enum" (with a space) is just an ordinary comment, not a directive, so
// it is not even flagged as malformed — it simply isn't found.
func findVowEnumDirective(doc *ast.CommentGroup) (rest string, ok bool, malformed bool, malformedText string) {
	if doc == nil {
		return "", false, false, ""
	}
	for _, c := range doc.List {
		if !strings.HasPrefix(c.Text, "//vow:") {
			continue
		}
		body := strings.TrimPrefix(c.Text, "//")
		switch {
		case body == "vow:enum":
			return "", true, false, ""
		case strings.HasPrefix(body, "vow:enum "):
			return strings.TrimSpace(strings.TrimPrefix(body, "vow:enum")), true, false, ""
		default:
			return "", false, true, c.Text
		}
	}
	return "", false, false, ""
}

// detectDetachedEnumDirective catches the gotcha spec section 7 calls the
// first thing every user hits: a blank line between a //vow:enum directive
// and the type declaration below it detaches the comment in Go's doc-comment
// association rule, so the directive silently does nothing. usedDocs holds
// every CommentGroup that legitimately is some declaration's Doc; anything
// else in the file containing a "//vow:enum" line is a directive the user
// wrote but that never attached to anything.
func detectDetachedEnumDirective(fset *token.FileSet, f *ast.File, usedDocs map[*ast.CommentGroup]bool) error {
	for _, cg := range f.Comments {
		if usedDocs[cg] {
			continue
		}
		for _, c := range cg.List {
			if strings.HasPrefix(c.Text, "//vow:enum") {
				return fmt.Errorf("%s: //vow:enum directive is not attached to any type declaration; in Go, a blank line between a doc comment and the declaration it documents detaches it — remove the blank line so this directive immediately precedes `type X string` with no gap",
					fset.Position(c.Pos()))
			}
		}
	}
	return nil
}

// collectConstMembers walks every const block in files and, for each typed
// const whose type matches a candidate enum's name, appends it as a member.
// It honors Go's const spec-repetition rule: a ValueSpec with no explicit
// Type inherits the previous spec's Type within the same GenDecl.
func collectConstMembers(fset *token.FileSet, files []*ast.File, candidates map[string]*enumType) error {
	for _, f := range files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			var lastType ast.Expr
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				if vs.Type != nil {
					lastType = vs.Type
				}
				if lastType == nil {
					continue
				}
				typeName := types.ExprString(lastType)
				enum, ok := candidates[typeName]
				if !ok {
					continue
				}
				if len(vs.Values) != len(vs.Names) {
					return errAt(fset.Position(vs.Pos()), enum.Name, "const %s must assign an explicit string literal to be an enum member", vs.Names[0].Name)
				}
				for i, nm := range vs.Names {
					if nm.Name == "_" {
						continue
					}
					lit, ok := vs.Values[i].(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						return errAt(fset.Position(vs.Values[i].Pos()), enum.Name, "const %s must be assigned a string literal to be an enum member", nm.Name)
					}
					val, err := strconv.Unquote(lit.Value)
					if err != nil {
						return errAt(fset.Position(lit.Pos()), enum.Name, "invalid string literal for const %s", nm.Name)
					}
					enum.Members = append(enum.Members, enumMember{ConstName: nm.Name, Value: val})
				}
			}
		}
	}
	return nil
}

// importCollector accumulates the extra (non-vow-runtime) imports a
// generated file needs, deduplicated by path, detecting and rejecting the
// case where two source files alias the same import path differently.
type importCollector struct {
	seen map[string]importSpec
}

func newImportCollector() *importCollector {
	return &importCollector{seen: map[string]importSpec{}}
}

func (c *importCollector) add(local, path string) error {
	want := importSpec{Path: path}
	if local != defaultLocalName(path) {
		want.Alias = local
	}
	if existing, ok := c.seen[path]; ok {
		if existing.Alias != want.Alias {
			return fmt.Errorf("package %q is imported under inconsistent aliases (%q and %q) across files in this directory; use one alias consistently so the generated file can import it once",
				path, aliasOrDefault(existing, path), aliasOrDefault(want, path))
		}
		return nil
	}
	c.seen[path] = want
	return nil
}

func aliasOrDefault(spec importSpec, path string) string {
	if spec.Alias != "" {
		return spec.Alias
	}
	return defaultLocalName(path)
}

func (c *importCollector) sorted() []importSpec {
	paths := make([]string, 0, len(c.seen))
	for p := range c.seen {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	out := make([]importSpec, 0, len(paths))
	for _, p := range paths {
		out = append(out, c.seen[p])
	}
	return out
}
