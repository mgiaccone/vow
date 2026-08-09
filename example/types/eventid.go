package types

import (
	"fmt"
	"regexp"
	"sync/atomic"

	"github.com/mgiaccone/vow"
)

var eventIDPattern = regexp.MustCompile(`^evt_[0-9a-f]{8}$`)

var eventIDSpec = vow.Spec[string]{
	Rules: []vow.Rule[string]{
		vow.NotBlank,
		vow.Matches(eventIDPattern, "must be an event id"),
	},
}

var eventIDCounter atomic.Uint32

// eventIDGenerator is found by name, the same way eventIDSpec is. Declaring
// it is the whole opt-in: it is what makes GenerateEventID exist. A real one
// would use a UUID; a counter keeps the example dependency-free and its
// output predictable in tests.
//
// It takes no parameters, which the generator enforces — a value that needs
// an argument to be built is not one vow can mint on its own.
func eventIDGenerator() string {
	return fmt.Sprintf("evt_%08x", eventIDCounter.Add(1))
}

// EventID is minted rather than parsed from input. It still has NewEventID
// and MustEventID, for reading one back out of a database or a request.
type EventID struct {
	v string `vow:"json,sql,text"`
}
