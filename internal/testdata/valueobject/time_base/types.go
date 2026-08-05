package fixture

import (
	"time"

	"github.com/mgiaccone/vow"
)

var expirySpec = vow.Spec[time.Time]{}

type Expiry struct {
	v time.Time `vow:"json,text"`
}
