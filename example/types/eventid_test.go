package types_test

import (
	"testing"

	"github.com/mgiaccone/vow/example/types"
)

// TestGenerateEventID_ProducesAValidValue is the point of a generator: no raw
// string at the call site, and no error to handle that cannot happen.
func TestGenerateEventID_ProducesAValidValue(t *testing.T) {
	id := types.GenerateEventID()

	if id.IsZero() {
		t.Fatal("expected a value")
	}
	// Round-trips through the ordinary constructor, which is what proves the
	// generator agrees with the spec rather than merely being trusted.
	back, err := types.NewEventID(id.Unwrap())
	if err != nil {
		t.Fatalf("a generated value failed its own spec: %v", err)
	}
	if back != id {
		t.Fatalf("round-trip changed the value: %v != %v", back, id)
	}
}

func TestGenerateEventID_ProducesDistinctValues(t *testing.T) {
	seen := make(map[types.EventID]struct{}, 100)
	for range 100 {
		id := types.GenerateEventID()
		if _, dup := seen[id]; dup {
			t.Fatalf("generated %v twice", id)
		}
		seen[id] = struct{}{}
	}
}

// TestEventID_StillParses: a generator adds Generate<T>, it does not replace
// the parsing constructors — an id read back from storage still arrives as a
// string that has to be checked.
func TestEventID_StillParses(t *testing.T) {
	if _, err := types.NewEventID("not-an-event-id"); err == nil {
		t.Fatal("expected malformed input to be rejected")
	}
	if _, err := types.NewEventID("evt_0000002a"); err != nil {
		t.Fatalf("expected a well-formed id to parse: %v", err)
	}
}
