package main

import (
	"fmt"
	"go/token"
)

// genError is a generation error carrying the position, the type it
// concerns, and an actionable message. Every rejection the generator makes
// produces one of these, so "vow: " errors are consistently formatted and
// consistently say what to do next.
type genError struct {
	pos  token.Position
	kind string // the type or enum name this concerns; "" if not type-specific
	msg  string
}

func (e *genError) Error() string {
	if e.kind == "" {
		return fmt.Sprintf("%s: %s", e.pos, e.msg)
	}
	return fmt.Sprintf("%s: type %s: %s", e.pos, e.kind, e.msg)
}

func errAt(pos token.Position, typeName, format string, args ...any) error {
	return &genError{pos: pos, kind: typeName, msg: fmt.Sprintf(format, args...)}
}
