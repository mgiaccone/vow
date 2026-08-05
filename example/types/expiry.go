package types

import (
	"errors"
	"time"

	"github.com/mgiaccone/vow"
)

// mustBeFuture rejects a time.Time that is not strictly after the moment
// it's checked. It's handwritten rather than built in because vow's
// generic rules (InRange, Positive) require cmp.Ordered, which time.Time
// doesn't satisfy — it has no < operator. That's not a gap to patch: rules
// stay as ordinary, type-checked Go rather than a fixed vocabulary the
// runtime would need to keep growing to cover every base type.
func mustBeFuture(t time.Time) error {
	if !t.After(time.Now()) {
		return errors.New("must be in the future")
	}
	return nil
}

var expirySpec = vow.Spec[time.Time]{
	Rules: []vow.Rule[time.Time]{mustBeFuture},
}

// Expiry is a value object for a timestamp that must be in the future at
// the moment it's parsed. It exercises a qualified base type: the
// generator resolves the "time" import for the generated methods from
// this file's own import table.
type Expiry struct {
	v time.Time `vow:"json,text"`
}
