package fixture

import "github.com/mgiaccone/vow"

var idSpec = vow.Spec[string]{Rules: []vow.Rule[string]{vow.NotBlank}}

// Found by name, exactly as idSpec is. Declaring it is what makes GenerateID
// appear; a type with no such func generates what it always did.
func idGenerator() string { return "generated" }

type ID struct {
	v string `vow:"json"`
}
