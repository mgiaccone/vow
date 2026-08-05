package main

import "go/token"

// baseKind classifies a value object's base type for the purpose of
// generating String, MarshalText, Value, and Scan. It is derived purely
// syntactically from the base type's expression string — never from
// go/types — because the generator must work on a package that does not
// type-check.
type baseKind int

const (
	kindString       baseKind = iota
	kindIntWidenable          // int, int8, int16, int32, uint8, uint16, uint32: fits losslessly in int64
	kindInt64                 // int64: already the type driver.Value wants
	kindUintWide              // uint, uint64: may not fit in int64; sql is rejected for these
	kindFloat                 // float32, float64
	kindBool
	kindOther // a qualified or other named type, e.g. time.Time
)

var builtinBaseKinds = map[string]baseKind{
	"string":  kindString,
	"int":     kindIntWidenable,
	"int8":    kindIntWidenable,
	"int16":   kindIntWidenable,
	"int32":   kindIntWidenable,
	"uint8":   kindIntWidenable,
	"uint16":  kindIntWidenable,
	"uint32":  kindIntWidenable,
	"int64":   kindInt64,
	"uint":    kindUintWide,
	"uint64":  kindUintWide,
	"float32": kindFloat,
	"float64": kindFloat,
	"bool":    kindBool,
}

// classifyBase returns the baseKind for a base type's syntactic expression
// string. Anything not a recognized Go builtin scalar — including qualified
// names like time.Time and any other named type — is kindOther.
func classifyBase(expr string) baseKind {
	if k, ok := builtinBaseKinds[expr]; ok {
		return k
	}
	return kindOther
}

// valueObject is the resolved, generator-ready description of one
// vow-tagged struct.
type valueObject struct {
	Name       string // "Email"
	Pos        token.Position
	BaseType   string // syntactic expression, e.g. "string", "int32", "time.Time"
	BaseKind   baseKind
	FieldName  string   // the struct's single unexported field, e.g. "v"
	SpecVar    string   // "emailSpec", derived or from spec=
	Sanitizers []string // exported vow sanitizer names in tag order, e.g. ["Trim", "Lower"]
	HasJSON    bool
	HasSQL     bool
	HasText    bool
}

// These five predicates classify BaseKind for the template, which branches
// on them to choose String, MarshalText, Value, and Scan bodies without
// embedding go/types logic in template text.

// IsString reports whether the base type is string.
func (v *valueObject) IsString() bool { return v.BaseKind == kindString }

// IsOther reports whether the base type is a qualified or other named
// type, e.g. time.Time, rather than a Go builtin scalar.
func (v *valueObject) IsOther() bool { return v.BaseKind == kindOther }

// IsBool reports whether the base type is bool.
func (v *valueObject) IsBool() bool { return v.BaseKind == kindBool }

// IsFloat reports whether the base type is float32 or float64.
func (v *valueObject) IsFloat() bool { return v.BaseKind == kindFloat }

// IsIntFamily reports whether the base type is one of the integer kinds
// the template widens to int64 for driver.Value: int, int8, int16, int32,
// int64, uint8, uint16, or uint32.
func (v *valueObject) IsIntFamily() bool {
	return v.BaseKind == kindIntWidenable || v.BaseKind == kindInt64
}

// ParserVar is the name of the package-level composed parser emitted when
// Sanitizers is non-empty: SpecVar.Sanitizing(...). Empty when there are no
// sanitizers, in which case generated code calls SpecVar.Parse directly.
func (v *valueObject) ParserVar() string {
	if len(v.Sanitizers) == 0 {
		return ""
	}
	return lowerLeadingRun(v.Name) + "Parser"
}

// enumMember is one typed const belonging to an enum's declared type.
type enumMember struct {
	ConstName string // "RoleOwner"
	Value     string // "owner", unquoted
}

// enumType is the resolved, generator-ready description of one
// //vow:enum-directed defined type.
type enumType struct {
	Name       string
	Pos        token.Position
	Members    []enumMember
	Sanitizers []string
	HasJSON    bool
	HasSQL     bool
	HasText    bool
}

// ParserVar is always emitted for an enum: there is no user-authored Spec
// to call directly, since spec= is rejected on //vow:enum.
func (e *enumType) ParserVar() string {
	return lowerLeadingRun(e.Name) + "Parser"
}

// ValuesVar is the private backing slice for <T>Values().
func (e *enumType) ValuesVar() string {
	return lowerLeadingRun(e.Name) + "Values"
}

// importSpec is one entry the generated file must import.
type importSpec struct {
	Alias string // "" for an unaliased import
	Path  string
}

// pkg is everything needed to render one output file.
type pkg struct {
	Name          string
	VowImportPath string
	VowQualifier  string
	Imports       []importSpec // extra imports beyond the vow runtime, sorted by Path
	ValueObjects  []*valueObject
	Enums         []*enumType
}
