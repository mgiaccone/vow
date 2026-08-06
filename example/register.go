package example

import (
	"github.com/mgiaccone/vow"
	"github.com/mgiaccone/vow/example/types"
)

const (
	FieldNumberType vow.Field = "Type"
	FieldNumber     vow.Field = "Number"
)

// RegisterNumber is the write path: the payload carries both the kind and
// the number, so the number can be validated against its kind.
type RegisterNumber struct {
	Type   types.PhoneNumberType
	Number types.TypedNumber
}

// NewRegisterNumber shows a field whose validity depends on another field.
// The kind is parsed first; the number is parsed with it, and only once the
// kind is known good.
func NewRegisterNumber(rawType, rawNumber string) (RegisterNumber, error) {
	var c vow.Collector

	cmd := RegisterNumber{
		Type: vow.Collect(&c, FieldNumberType, rawType, types.NewPhoneNumberType),
	}

	// The number cannot go in the literal above: it needs the kind. Guarding
	// on c.OK keeps a bad kind from producing a second, misleading error
	// here from typedNumberSpec's fail-closed default — the real complaint
	// is about the kind, and it is already recorded.
	if c.OK(FieldNumberType) {
		cmd.Number = vow.CollectFunc(&c, FieldNumber, func() (types.TypedNumber, error) {
			return types.NewTypedNumber(rawNumber, cmd.Type)
		})
	}

	return cmd, c.Err()
}

// LookupNumber is the read path: the request carries a number but no kind,
// so the number can only be checked as well-formed for some kind. Its field
// is an AnyNumber, which cannot be passed where a TypedNumber is required.
type LookupNumber struct {
	Number types.AnyNumber
}

func NewLookupNumber(rawNumber string) (LookupNumber, error) {
	var c vow.Collector
	q := LookupNumber{
		Number: vow.Collect(&c, FieldNumber, rawNumber, types.NewAnyNumber),
	}
	return q, c.Err()
}
