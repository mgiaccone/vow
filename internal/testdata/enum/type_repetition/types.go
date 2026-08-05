package fixture

// Status exercises Go's const spec-repetition rule: only the first spec in
// the block states its type explicitly, and the rest inherit it.
//
//vow:enum
type Status string

const (
	StatusOpen   Status = "open"
	StatusClosed        = "closed"
	StatusMerged        = "merged"
)
