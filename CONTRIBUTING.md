# Contributing to vow

Issues and pull requests are welcome.

## Getting started

```
git clone https://github.com/mgiaccone/vow
cd vow
go test ./...
```

No setup beyond a Go 1.24+ toolchain. `vow` has zero third-party
dependencies, so there is nothing to install.

## Before opening a pull request

```
go vet ./...
gofmt -l .            # must print nothing
go test ./...
```

If you changed the generator, also confirm the committed output is current:

```
go generate ./...
git diff --exit-code
```

CI runs exactly these, so a green local run means a green build.

## Changing the generator

Generated output is checked into `example/` and `internal/testdata/`, so any
generator change shows up as a golden-file diff. The workflow:

1. `go test ./cmd/vow -update` to refresh the golden files.
2. **Read the diff by eye.** Don't accept it because the tests pass — the
   tests compare against whatever you just wrote. You are reviewing the
   generated Go, not the diff's size.
3. `go generate ./...` and confirm `example/` still builds and vets.
4. Run generation twice and confirm the second run produces no diff. Output
   must be byte-identical across runs, or the CI freshness check becomes a
   source of spurious failures.

## Invariants

Some constraints in this codebase look arbitrary until you know why they
exist. [`CLAUDE.md`](CLAUDE.md) lists each one with its reason — read that
before working around one. The load-bearing ones:

- **Zero third-party dependencies**, including in tests. This is the whole
  pitch to anyone weighing a codegen tool; one `require` line undoes it.
- **No type-checking in the generator.** Discovery uses `go/parser` and
  syntactic checks only, because the generator must work on a package that
  doesn't compile yet — that's the normal state during first adoption, when
  the code calls `NewEmail` before `NewEmail` exists.
- **`vow` never deletes a file.** It writes exactly one path and touches
  nothing else.
- **Deterministic output.** Sorted iteration everywhere that reaches disk.

## Proposing new features

Open an issue first for anything that changes the public API, adds a struct
tag or directive option, or adds a dependency. The generated method set and
the built-in rule set are deliberately small; additions need a case, not just
an implementation.

Several things are deliberately out of scope — struct validation,
`UnmarshalJSON`, validation inside `Scan`, multi-field value objects, and
rules expressed in struct tags. The README's [Gotchas][g] and [What Go will
not let this library do][w] sections explain the reasoning for each.

[g]: README.md#gotchas
[w]: README.md#what-go-will-not-let-this-library-do

## Commit messages

Describe what changed and why. The "why" matters more — the diff already
shows the what.

## License

Contributions are licensed under the [MIT License](LICENSE).
