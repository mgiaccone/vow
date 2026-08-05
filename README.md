# vow

[![Go Reference](https://pkg.go.dev/badge/github.com/mgiaccone/vow.svg)](https://pkg.go.dev/github.com/mgiaccone/vow)
[![CI](https://github.com/mgiaccone/vow/actions/workflows/ci.yml/badge.svg)](https://github.com/mgiaccone/vow/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> **vow** — value objects for Go. The name is a promise the value is valid;
> it also happens to stand for *Validated On Write*, which is the rule the
> whole design follows.

## The problem

Most Go code re-validates the same string or int every time it crosses a
boundary — once in the HTTP handler, again before the SQL insert, again in
the background job that reads it back. Each check is a slightly different
regex, a slightly different message, and a slightly different bug, because
nothing stops a value from existing without ever having passed one. `vow`
moves validation to the one place it actually needs to happen: construction.
A `vow.Spec` sanitizes and validates a value, and the generator turns it into
a `New<T>` constructor that's the *only* way to produce a `T` — so once
you're holding an `Email`, it is already valid, and stays that way for its
entire life. It is not a struct validator: `vow` never reads a request struct
or a `json` tag, only the individual values that go into one.

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
- [Prior art and positioning](#prior-art-and-positioning)
- [Contributing](#contributing)
- [License](#license)

## Install

`vow` requires **Go 1.24 or later** — using it as a tool dependency needs
1.24 either way, so the module's own floor doesn't add a separate
constraint. The recommended path is a pinned tool dependency:

```
go get -tool github.com/mgiaccone/vow/cmd/vow@v1.0.0
```

That adds a `tool` directive to your `go.mod`, separate from `require`. Then
add a generate directive next to the types you're generating for:

```go
//go:generate go tool vow -dir=.
```

`go tool` resolves the last path segment, so the invocation is just `vow`.
The first run compiles the binary; later runs are cached, and contributors
need nothing installed globally — `go generate ./...` after a clone just
works, at the version this repository pins.

The wart worth knowing about: tool dependencies share your module's
dependency graph, so adding one *can* bump a version your application code
also uses. `vow` has zero dependencies of its own, so adding it can never
perturb anything else. That's the whole reason the zero-dependency
constraint exists — feel it here.

If you'd rather not add a tool dependency, running it directly still works:

```go
//go:generate go run github.com/mgiaccone/vow/cmd/vow@v1.0.0 -dir=.
```

This needs a `require` entry (`go mod tidy` adds one). Avoid `@latest` in a
generate directive — generated code that changes because upstream released
while you were at lunch isn't reproducible, and it breaks the `git diff
--exit-code` CI check below. Prefer `go tool` when you can.

This repository dogfoods the `tool` path for its own `example/` package —
see this module's own `go.mod`.

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

The spec variable name is derived from the type name if `spec=` is absent:
lowercase the leading run of capitals, keeping the capital that starts the
next word, and append `Spec`. `Email` → `emailSpec`, `URLPath` →
`urlPathSpec`, `URL` → `urlSpec`.

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
| `MaxLen(n)` | `ErrTooLong` | must be at most `n` characters |
| `MinLen(n)` | `ErrTooShort` | must be at least `n` characters |
| `Matches(re, msg)` | `ErrNotMatch` | `msg` |
| `OneOf(...)` | `ErrNotInSet` | must be one of: ... |
| `InRange(lo, hi)` | `ErrOutOfRange` | must be between `lo` and `hi` |
| `Positive` | `ErrOutOfRange` | must be greater than zero |

**CLI flags** (`cmd/vow`):

| Flag | Default | Meaning |
|---|---|---|
| `-dir` | `.` | Directory to scan. |
| `-out` | `zz_generated_vow.go` | Output file name, written inside `-dir`. |
| `-vow-qualifier` | `vow` | Local qualifier for the runtime import in generated code. |
| `-tag-key` | `vow` | Struct tag key that marks a value object field. |
| `-v` | off | Print a one-line summary on success. |

Exit code 0 on success, 1 on any error; errors go to stderr prefixed `vow: `.
Output is one file per package — not configurable, and never deleted.

## Reporting several failures at once

`Spec.Parse` is fail-fast: one value, one reason. Reporting several fields at
once — a command with three inputs, say — is `vow.Collector`'s job:

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
returning the zero value on failure — check `c.Err()` (or, on a generated
type, `IsZero`) before trusting the result. `vow.FieldErrors(err)` walks the
joined error and returns every `FieldError` in it — use this instead of
`errors.As`, which stops at the first match in a joined error and silently
drops the rest. See `example/invite.go` for the full constructor, including
the one rule a per-type generator can't express on its own: the invitee must
not be the inviter.

Mapping a `vow.Field` to a wire name is the last hop, and it belongs in your
transport adapter — the only layer that knows the wire format. Illustration,
not part of vow's API:

```go
// Illustration only. Mapping vow.Field to wire names is the transport
// adapter's job, not vow's — it's the only layer that knows the format.
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
so a decode failure can't identify which field broke, and can't join a
`Collector` result. Decode into a string DTO and call `New<T>` where the
field is known.

**`Scan` does not validate.** If a rule is later tightened or an enum member
retired, re-validating on read would make historical rows unloadable —
including the rows you'd need to load to fix them. Validate at the boundary,
trust storage.

**Enum conversions still compile.** `Role("wizard")` is legal Go from
anywhere; nothing can prevent it. That conversion is your own code, written
deliberately at a site you can grep for — not untrusted input, the way an
invalid `Email` string is.

**Single-field value objects only.** `Money{amount, currency}` needs
cross-field rules and a multi-argument constructor, which `New<T>(in Base)`
can't express. Hand-write those with `vow.Collector`, the same way
`example/invite.go` hand-writes `inviter != invitee`.

## What Go will not let this library do

These aren't shortcomings to work around later — they're properties of the
language, with the reason vow does what it does instead.

- **Every type has a zero value, and nothing can forbid it.** `var e Email`
  is legal Go and never passes through a constructor. That's why every
  generated type has `IsZero` (every enum, `IsValid`), and why a failed
  `Collect` returns the zero value rather than something safer.
- **Conversions to a defined type are always legal.** Go has no way to
  restrict them, so enums trade unforgeability for real `const` members and
  working exhaustiveness linters.
- **Unexported means package-scoped, not type-scoped.** Inside `package
  types`, `Email{v: "garbage"}` compiles. Keep the types package free of code
  with a motive to bypass a constructor, rather than trying to isolate each
  type in its own package.
- **There are no typed struct constants.** A struct-wrapped value object can
  never be a `const`, which is the whole reason enums use defined string
  types while value objects use struct wrappers — not a stylistic
  inconsistency.
- **Generics can't produce distinct nominal types.** A single
  `Validated[T any]` would make `Validated[string]` the same type for an
  email and a display name. This is the actual answer to "why does this need
  codegen at all" — Go has no way to derive a new named type from a generic
  instantiation.
- **`UnmarshalJSON` has nowhere to report which field failed.** A limit of
  the interface, not a preference — see Gotchas above.
- **Struct tags are backtick literals.** No embedded quotes, nothing
  type-checked, `,`-splitting corrupts any regex containing one. Rules stay
  as ordinary Go for exactly this reason.
- **Doc directives detach on a blank line.** See Gotchas above.
- **Unexported fields are invisible to reflection-based libraries.** The
  generated `MarshalJSON`/`MarshalText`/`Value`/`Scan` cover the common
  paths; a library that reaches for struct fields directly will see nothing.

## Design notes

**Rules are Go, not tag syntax.** A parser for a validation DSL embedded in a
struct tag has to be quote-aware to survive a regex containing a comma, and
it costs you go-to-definition, rename refactoring, and named-constant
references along the way. Ordinary `vow.Rule[T]` functions keep all of that.
See [`Rule`](https://pkg.go.dev/github.com/mgiaccone/vow#Rule).

**Value objects are struct wrappers; enums are defined types.** Forced by
"there are no typed struct constants" above — the only way to get real
`const` members and working exhaustiveness linters is a defined type, and the
only way to make a wrapped value unforgeable is a struct with an unexported
field.

**One file per package, not configurable.** Spec vars and enum consts are
already resolved across the whole package, so per-file output would need the
generator to delete files it no longer believes it owns — dangerous in a
tool whose whole pitch is that it's easy to trust. A single always-rewritten
path makes stale output structurally impossible.

## Prior art and positioning

[`go-playground/validator`](https://github.com/go-playground/validator) and
[`ozzo-validation`](https://github.com/go-ozzo/ozzo-validation) validate
*structs at a boundary*. `vow` makes *values* unforgeable so there's no
boundary left to re-check.

[`go-enum`](https://github.com/abice/go-enum) is the closest neighbor on the
enum side. `vow` covers value objects, sanitizers, and enums with one tool,
and rules stay ordinary Go rather than generator-specific syntax.

[`nao1215/vogen`](https://github.com/nao1215/vogen) solves the same value-
object problem and will come up in every comparison. It declares value
objects as metadata in a separate `main.go` and generates getters,
constructors, and `Equal()` from that. `vow` declares them at the type site
with a struct tag and a `Spec` of ordinary Go rules, and adds sanitizers and
enums. Neither is a strict superset of the other. `vo` is vogen's default
generated package name; `vow` is deliberately adjacent but distinct, and no
relationship between the two projects should be inferred from that.

## Contributing

Issues and PRs welcome. Before sending one: `go vet ./... && gofmt -l .`,
`go test ./...`, and if you touched the generator, `go generate ./...`
followed by `git diff --exit-code` to confirm committed output is current.
See [`CLAUDE.md`](CLAUDE.md) for the invariants a change needs to respect.

## License

[MIT](LICENSE)
