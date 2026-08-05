package main

import (
	"go/token"
	"strings"
)

// tagToken is one comma-separated option from a vow struct tag or a
// //vow:enum directive, before it is checked against the known option set.
type tagToken struct {
	Key      string
	Value    string
	HasValue bool
}

// parseTagOptions splits raw on commas and each part on the first '='.
// Surrounding whitespace is trimmed from every token, so both a struct
// tag's tight "sanitize=trim|lower,json" and a directive's looser
// "sanitize=trim|lower, json, sql" parse identically. Stray blank entries
// (a trailing comma, doubled commas) are tolerated and dropped.
func parseTagOptions(raw string) []tagToken {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	tokens := make([]tagToken, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if idx := strings.IndexByte(p, '='); idx >= 0 {
			tokens = append(tokens, tagToken{Key: strings.TrimSpace(p[:idx]), Value: strings.TrimSpace(p[idx+1:]), HasValue: true})
		} else {
			tokens = append(tokens, tagToken{Key: p})
		}
	}
	return tokens
}

// knownSanitizers maps a tag's lowercase sanitizer spelling to the exported
// vow function name. This set is deliberately closed — see vow.Trim's
// godoc for why — so an unrecognized name is always a rejection, never a
// silent no-op.
var knownSanitizers = map[string]string{
	"trim":     "Trim",
	"lower":    "Lower",
	"upper":    "Upper",
	"collapse": "Collapse",
}

// options is the validated result of resolving a vow struct tag or
// //vow:enum directive against the known option set.
type options struct {
	Sanitizers   []string // exported vow identifiers, in tag order
	SpecOverride string
	HasSpec      bool
	HasJSON      bool
	HasSQL       bool
	HasText      bool
}

// resolveOptions validates tokens against the known option set: sanitize=,
// spec= (value objects only), and the bare flags json, sql, text.
// allowSpec is false for //vow:enum, where spec= is rejected outright:
// enum membership is generated from the const block, so there is no
// user-authored Spec for spec= to name.
func resolveOptions(tokens []tagToken, allowSpec bool, pos token.Position, typeName string) (options, error) {
	var out options
	for _, tok := range tokens {
		switch tok.Key {
		case "sanitize":
			if !tok.HasValue || tok.Value == "" {
				return out, errAt(pos, typeName, "sanitize= requires a value, e.g. sanitize=trim|lower")
			}
			for _, name := range strings.Split(tok.Value, "|") {
				name = strings.TrimSpace(name)
				fn, ok := knownSanitizers[name]
				if !ok {
					return out, errAt(pos, typeName, "unknown sanitizer %q; known sanitizers are trim, lower, upper, collapse", name)
				}
				out.Sanitizers = append(out.Sanitizers, fn)
			}
		case "spec":
			if !allowSpec {
				return out, errAt(pos, typeName, "spec= is not valid on //vow:enum; enum membership is generated from the const block, so there is no user-authored Spec to reference")
			}
			if !tok.HasValue || tok.Value == "" {
				return out, errAt(pos, typeName, "spec= requires a value, e.g. spec=emailSpec")
			}
			out.SpecOverride = tok.Value
			out.HasSpec = true
		case "json":
			if tok.HasValue {
				return out, errAt(pos, typeName, "json does not take a value in this version")
			}
			out.HasJSON = true
		case "sql":
			if tok.HasValue {
				return out, errAt(pos, typeName, "sql does not take a value in this version")
			}
			out.HasSQL = true
		case "text":
			if tok.HasValue {
				return out, errAt(pos, typeName, "text does not take a value in this version")
			}
			out.HasText = true
		default:
			return out, errAt(pos, typeName, "unknown tag option %q", tok.Key)
		}
	}
	return out, nil
}
