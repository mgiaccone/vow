package example

import (
	"github.com/mgiaccone/vow"
	"github.com/mgiaccone/vow/example/types"
)

// Field constants for SendBroadcast. FieldRecipients names the whole list,
// not any one element: CollectSlice keeps element positions inside the error
// rather than in the field, so this stays usable as a c.OK guard and as a key
// in a wire-name map.
const (
	FieldSender     vow.Field = "Sender"
	FieldRecipients vow.Field = "Recipients"
)

// SendBroadcast is a command with a list-valued field.
type SendBroadcast struct {
	Sender     types.Email
	Recipients []types.Email
}

// NewSendBroadcast parses a sender and a list of recipients, reporting every
// bad address in one pass. CollectSlice applies the same constructor to each
// element, so the list is either wholly valid or nil — it never comes back
// quietly shortened, which would broadcast to a subset of the intended
// audience.
//
// Recipients are checked with NoDuplicates rather than Deduped: mailing the
// same person twice is a mistake the caller should hear about, not something
// to paper over. A tag list would want the opposite.
//
// The duplicate check runs on parsed values, which is the only place it can
// work. "  Boss@Example.com " and "boss@example.com" are different strings
// but the same Email once Email's sanitizers have run.
func NewSendBroadcast(rawSender string, rawRecipients []string) (SendBroadcast, error) {
	var c vow.Collector
	cmd := SendBroadcast{
		Sender:     vow.Collect(&c, FieldSender, rawSender, types.NewEmail),
		Recipients: vow.CollectSlice(&c, FieldRecipients, rawRecipients, types.NewEmail, vow.NoDuplicates),
	}
	return cmd, c.Err()
}
