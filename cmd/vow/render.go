package main

import (
	"bytes"
	"fmt"
	"go/format"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

var vowSourceTemplate = template.Must(template.New("vow").Funcs(template.FuncMap{
	"quotedMembers": quotedMembers,
}).Parse(sourceTemplateText))

// render turns a resolved *pkg into formatted Go source. It always runs the
// template output through format.Source: that gofmts the result, and a
// malformed template produces a clear parse error here instead of a
// mystery compile error in the consumer's build.
func render(p *pkg) ([]byte, error) {
	data := struct {
		*pkg
		AllImports []importSpec
	}{pkg: p, AllImports: buildImportList(p)}

	var buf bytes.Buffer
	if err := vowSourceTemplate.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("formatting generated source: %w\n\n--- unformatted source ---\n%s", err, buf.String())
	}
	return formatted, nil
}

// buildImportList computes every import the output file needs beyond the
// vow runtime — encoding/json, database/sql/driver, fmt, and slices are
// each included only if something in p actually uses them — merges in the
// qualified-base-type imports parse.go already resolved, adds the vow
// runtime import itself, and sorts the result by path.
func buildImportList(p *pkg) []importSpec {
	needsJSON := false
	needsSQL := false
	needsFmt := false
	needsSlices := len(p.Enums) > 0

	for _, vo := range p.ValueObjects {
		if vo.HasJSON {
			needsJSON = true
		}
		if vo.HasSQL {
			needsSQL = true
			needsFmt = true // Scan's error path uses fmt.Errorf
		}
		if vo.BaseKind != kindString {
			needsFmt = true // String falls back to fmt.Sprint
		}
	}
	for _, e := range p.Enums {
		if e.HasJSON {
			needsJSON = true
		}
		if e.HasSQL {
			needsSQL = true
			needsFmt = true
		}
	}

	var list []importSpec
	if needsSQL {
		list = append(list, importSpec{Path: "database/sql/driver"})
	}
	if needsJSON {
		list = append(list, importSpec{Path: "encoding/json"})
	}
	if needsFmt {
		list = append(list, importSpec{Path: "fmt"})
	}
	if needsSlices {
		list = append(list, importSpec{Path: "slices"})
	}
	list = append(list, p.Imports...)

	vowImport := importSpec{Path: p.VowImportPath}
	if p.VowQualifier != defaultLocalName(p.VowImportPath) {
		vowImport.Alias = p.VowQualifier
	}
	list = append(list, vowImport)

	sort.Slice(list, func(i, j int) bool { return list[i].Path < list[j].Path })
	return list
}

func quotedMembers(members []enumMember) string {
	parts := make([]string, len(members))
	for i, m := range members {
		parts[i] = strconv.Quote(m.Value)
	}
	return strings.Join(parts, ", ")
}
