# vow

[![Go Reference](https://pkg.go.dev/badge/github.com/mgiaccone/vow.svg)](https://pkg.go.dev/github.com/mgiaccone/vow)
[![CI](https://github.com/mgiaccone/vow/actions/workflows/ci.yml/badge.svg)](https://github.com/mgiaccone/vow/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> **vow** — value objects for Go. Short for *Validated On Write*: a value is
> parsed once, at construction, and is unforgeable from then on.

## The problem

The same string gets re-validated every time it crosses a boundary: once in
the HTTP handler, again before the SQL insert, again in the job that reads it
back. Nothing stops a value from existing without having passed any of them,
so the checks drift apart and each becomes its own bug.

`vow` moves validation to construction. A `vow.Spec` sanitizes and validates,
and the generator turns it into the only constructor that can produce the
type — so holding an `Email` means it is already valid.

It is not a struct validator: `vow` never reads your request structs or their
`json` tags, only the individual values that go into them.

## Table of contents

- [Install](#install)
- [Quickstart: a value object](#quickstart-a-value-object)
- [Quickstart: an enum](#quickstart-an-enum)
- [What gets generated](#what-gets-generated)
- [Reference](#reference)
- [Reporting several failures at once](#reporting-several-failures-at-once)
- [Gotchas](#gotchas)
- [What Go will not let this library do](#what-go-will-not-let-this-library-do)
- [Design notes](#design-notes)
- [Alternatives](#alternatives)
- [Contributing](#contributing)

## Install

Requires **Go 1.24+**. Add `vow` as a tool dependency:

```
go get -tool github.com/mgiaccone/vow/cmd/vow@latest
```

Then put a generate directive next to your types:

```go
//go:generate go tool vow -dir=.
```

`go tool` resolves the last path segment, so the invocation is just `vow`.
The first run compiles the binary, later runs are cached, and a fresh clone
needs nothing installed globally.

Tool dependencies share your module's dependency graph, so adding one can bump
a version your application code also uses. `vow` has no dependencies, so it
can't.

`go run` works too, but pin an explicit version — unlike `go get`, a generate
directive re-resolves on every run, so `@latest` there means generated code
that changes when upstream releases, breaking the CI check below:

```go
//go:generate go run github.com/mgiaccone/vow/cmd/vow@vX.Y.Z -dir=.
```

## Quickstart: a value object

```go
package types

import "github.com/mgiaccone/vow"

var emailSpec = vow.Spec[string]{
	Rules: []vow.Rule[string]{
		vow.NotBlank,
		vow.MaxLen(254),
	},
}

type Email struct {
	v string `vow:"sanitize=trim|lower,json,sql,text"`
}
```

`Email` finds its rules through the **name** `emailSpec`, not through
proximity or file layout — see [Naming](#naming). The two can live in
different files of the same package.

```go
//go:generate go tool vow -dir=.
```

After `go generate`:

```go
e, err := types.NewEmail("  USER@Example.com  ")
// e.String() == "user@example.com", err == nil

_, err = types.NewEmail("")
// err != nil, err.Error() == "is required"
```

## Quickstart: an enum

```go
// Role is a closed set of membership levels.
//
//vow:enum sanitize=trim|lower, json
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)
```

```go
r, err := NewRole("  ADMIN  ")
// r == RoleAdmin, err == nil

_, err = NewRole("wizard")
// err != nil, err.Error() == "must be one of: owner, admin, member"
```

## What gets generated

**Value object mode** — a struct with one `vow`-tagged field:

```go
func New<T>(in Base) (T, error)
func Must<T>(in Base) T             // panics; for tests and constants
func (x T) Unwrap() Base
func (x T) IsZero() bool            // x == T{}
func (x T) String() string          // direct return for string bases,
                                     // fmt.Sprint otherwise
func (x T) MarshalJSON() ([]byte, error)     // if json
func (x T) MarshalText() ([]byte, error)     // if text
func (x T) Value() (driver.Value, error)     // if sql
func (x *T) Scan(src any) error              // if sql
```

**Enum mode** — a defined string type with a `//vow:enum` directive:

```go
func <T>Values() []T                // slices.Clone of the private slice
func New<T>(in string) (T, error)
func Must<T>(in string) T
func (x T) IsValid() bool           // slices.Contains; zero value is NOT valid
func (x T) String() string
func (x T) Unwrap() string
func (x T) IsZero() bool
func (x T) MarshalJSON() ([]byte, error)     // if json
func (x T) MarshalText() ([]byte, error)     // if text
func (x T) Value() (driver.Value, error)     // if sql
func (x *T) Scan(src any) error              // if sql
```

`sql` emits the two `database/sql` interfaces and nothing else — no pgx,
sqlx, ent, or GORM variants, and none are needed. pgx's `pgtype` resolves its
encode and scan plans through `driver.Valuer` and `sql.Scanner` for types it
doesn't recognize, so `sql`-generated types work with native pgx unmodified,
including `NULL` scanning to the zero value and retired enum members still
loading. Verified against pgx v5.10.0; `example/types/types_test.go` locks in
the part `vow` controls — that the interfaces are satisfied — without
importing pgx, which would put a third-party dependency in this module.

The membership rule behind `IsValid` and `New<T>` is generated from the enum's
own `const` block, so the accepted set can never drift from the declared
members.

## Reference

**Struct tag options** (key `vow`, overridable with `-tag-key`):

| Option | Meaning |
|---|---|
| `sanitize=a\|b` | Apply sanitizers `a`, then `b`, before validating. String bases only. |
| `spec=NAME` | Use `var NAME` instead of the derived spec variable name. |
| `json` | Emit `MarshalJSON`. |
| `sql` | Emit `Value` and `Scan`. |
| `text` | Emit `MarshalText`. |

### Naming

A tagged type is matched to its `Spec` **by name**. With no `spec=` option,
`vow` lowercases the leading run of capitals — keeping the capital that starts
the next word — and appends `Spec`:

| Type | Expected spec variable |
|---|---|
| `Email` | `emailSpec` |
| `URLPath` | `urlPathSpec` |
| `URL` | `urlSpec` |
| `ID` | `idSpec` |

Four things to know:

- **The spec must be a package-level `var`.** Declared inside a function, or
  in a `_test.go` file, it won't be found — `vow` errors and names the
  variable it expected. It may live in any file of the package.
- **Its type parameter must match the base type.** `vow` never type-checks, so
  a `Spec[int]` paired with a `string` field generates without complaint and
  fails to compile afterwards, pointing at the generated file.
- **`spec=NAME` overrides the derivation**, which is how several types share
  one `Spec`.
- **Some identifiers are reserved.** Declaring one yourself is rejected with
  the position of your declaration, rather than left to surface as a
  redeclaration error inside the generated file:

  | Declaration | Reserved |
  |---|---|
  | any tagged type or enum | `New<T>`, `Must<T>` |
  | value object with `sanitize=` | `<name>Parser` (e.g. `emailParser`) |
  | enum | `<T>Values`, `<name>Parser`, `<name>Values` (e.g. `RoleValues`, `roleParser`, `roleValues`) |

**`//vow:enum` directive options:**

| Option | Meaning |
|---|---|
| `sanitize=a\|b` | Apply sanitizers before checking membership. |
| `json` | Emit `MarshalJSON`. |
| `sql` | Emit `Value` and `Scan`. |
| `text` | Emit `MarshalText`. |

`spec=` is not valid here — enum membership is generated from the `const`
block, so there is no user-authored `Spec` to name.

**Built-in sanitizers** (`sanitize=`, closed set):

| Tag name | Function |
|---|---|
| `trim` | `vow.Trim` |
| `lower` | `vow.Lower` |
| `upper` | `vow.Upper` |
| `collapse` | `vow.Collapse` |

**Built-in rules:**

| Function | Sentinel | Message |
|---|---|---|
| `NotBlank` | `ErrBlank` | is required |
| `NotZero` | `ErrBlank` | is required |
| `MaxLen(n)` | `ErrTooLong` | must be at most `n` characters |
| `MinLen(n)` | `ErrTooShort` | must be at least `n` characters |
| `Matches(re, msg)` | `ErrNotMatch` | `msg` |
| `OneOf(...)` | `ErrNotInSet` | must be one of: ... |
| `InRange(lo, hi)` | `ErrOutOfRange` | must be between `lo` and `hi` |
| `Positive` | `ErrOutOfRange` | must be greater than zero |
| `NonNegative` | `ErrOutOfRange` | must not be negative |

**CLI flags** (`cmd/vow`):

| Flag | Default | Meaning |
|---|---|---|
| `-dir` | `.` | Directory to scan. |
| `-out` | `<package>_vow_generated.go` | Output file name, written inside `-dir`. The default is derived from the package's own name. |
| `-vow-qualifier` | `vow` | Local qualifier for the runtime import in generated code. |
| `-tag-key` | `vow` | Struct tag key that marks a value object field. |
| `-v` | off | Print a one-line summary on success. |

Exit code 0 on success, 1 on any error; errors go to stderr prefixed `vow: `.
Output is one file per package — not configurable, and never deleted.

## Reporting several failures at once

`Spec.Parse` is fail-fast: one value, one reason. Reporting several fields at
once is `vow.Collector`'s job:

```go
var c vow.Collector
cmd := CreateInvite{
	Inviter: vow.Collect(&c, FieldInviter, inviter, types.NewEmail),
	Invitee: vow.Collect(&c, FieldInvitee, invitee, types.NewEmail),
	Role:    vow.Collect(&c, FieldRole, role, types.NewRole),
}
return cmd, c.Err()
```

`Collect` runs a constructor and records any failure against a `Field`,
returning the zero value on failure — check `c.Err()` before trusting the
result. For logic that spans fields, guard with `c.OK(FieldA, FieldB)`, which
reports whether those fields parsed. Don't guard with `IsZero`: for a type
whose zero value is valid it is true after a *successful* parse, so the guard
skips checks that should run. `vow.FieldErrors(err)` walks the joined error
and returns every `FieldError` in it; use it instead of `errors.As`, which
stops at the first match and silently drops the rest. `example/invite.go` has
the full constructor, including the rule a per-type generator can't express:
the invitee must not be the inviter.

Mapping a `vow.Field` to a wire name belongs in your transport adapter, the
only layer that knows the wire format. Illustration, not part of vow's API:

```go
var wireNames = map[vow.Field]string{
	FieldInviter: "inviter_email",
	FieldInvitee: "invitee_email",
	FieldRole:    "role",
}

func toHTTPErrors(err error) map[string]string {
	out := map[string]string{}
	for _, fe := range vow.FieldErrors(err) {
		name, ok := wireNames[fe.Field]
		if !ok {
			name = string(fe.Field)
		}
		out[name] = fe.Error()
	}
	return out
}
```

## Gotchas

**A blank line detaches `//vow:enum`.** Go's doc-comment association rule —
the same one that governs `//go:build` — requires the directive to
immediately precede the declaration, with no blank line between them:

```go
// Correct — directive is attached:
//vow:enum
type Role string

// Wrong — the blank line detaches it; Role is treated as a plain type, and
// vow reports this as an error naming the position, rather than silently
// generating nothing:
//vow:enum

type Role string
```

**No `UnmarshalJSON`.** Its signature carries a byte slice and nothing else,
so a decode failure can't say which field broke or join a `Collector` result.
Decode into a string DTO and call `New<T>` where the field is known.

**`Scan` does not validate.** Tightening a rule or retiring an enum member
would otherwise make historical rows unloadable — including the ones you'd
need to load to fix them. Validate at the boundary, trust storage.

**Enum conversions still compile.** `Role("wizard")` is legal from anywhere.
But that's your own code, written deliberately at a site you can grep for —
not untrusted input the way an invalid `Email` string is.

**Single-field value objects only.** `Money{amount, currency}` needs
cross-field rules and a multi-argument constructor, which `New<T>(in Base)`
can't express. Hand-write those with `vow.Collector`, as `example/invite.go`
does for `inviter != invitee`.

## What Go will not let this library do

Properties of the language, not shortcomings to work around later.

- **Every type has a zero value, and nothing can forbid it.** `var e Email`
  is legal and never passes through a constructor. Hence `IsZero` on every
  generated type, `IsValid` on every enum, and a failed `Collect` returning
  the zero value.
- **Conversions to a defined type are always legal.** Go can't restrict them,
  so enums trade unforgeability for real `const` members and working
  exhaustiveness linters.
- **Unexported means package-scoped, not type-scoped.** Inside `package
  types`, `Email{v: "garbage"}` compiles. Keep that package free of code with
  a motive to bypass a constructor, rather than isolating each type in its
  own package.
- **There are no typed struct constants.** A struct-wrapped value object can
  never be a `const`. This is why enums are defined string types while value
  objects are struct wrappers — the two need different representations to get
  `const` members and unforgeability respectively, and only a struct field
  can carry a tag, which is why enums need a directive instead.
- **Generics can't produce distinct nominal types.** A single
  `Validated[T any]` would make `Validated[string]` the same type for an email
  and a display name. This is why the library generates code instead of
  shipping one generic type.
- **Struct tags are backtick literals.** No embedded quotes, nothing
  type-checked, and `,`-splitting corrupts any regex containing one — which
  is why rules stay ordinary Go.
- **Unexported fields are invisible to reflection-based libraries.** The
  generated `MarshalJSON`/`MarshalText`/`Value`/`Scan` cover the common paths;
  a library reaching for struct fields directly sees nothing.

## Design notes

**Rules are Go, not tag syntax.** A validation DSL in a struct tag needs a
quote-aware parser to survive a regex containing a comma, and costs you
go-to-definition, rename refactoring, and named-constant references. Ordinary
[`Rule[T]`](https://pkg.go.dev/github.com/mgiaccone/vow#Rule) functions keep
all of that.

**One file per package, not configurable.** Spec vars and enum consts already
resolve across the whole package. Per-file output would mean the generator
deleting files it believes it owns — dangerous in a tool whose pitch is that
it's easy to trust. One always-rewritten path makes stale output impossible.

## Alternatives

- [`go-playground/validator`](https://github.com/go-playground/validator),
  [`ozzo-validation`](https://github.com/go-ozzo/ozzo-validation) — validate
  structs at a boundary. `vow` makes values unforgeable so there's no boundary
  left to re-check.
- [`go-enum`](https://github.com/abice/go-enum) — enums only, with rules in
  generator-specific syntax rather than ordinary Go.
- [`nao1215/vogen`](https://github.com/nao1215/vogen) — same problem from the
  other end: value objects declared as metadata in a separate `main.go`,
  rather than at the type site. Unrelated project.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
