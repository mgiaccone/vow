package fixture

import "github.com/mgiaccone/vow"

var emailSpec = vow.Spec[string]{
	Rules: []vow.Rule[string]{vow.NotBlank},
}

type Email struct {
	v string `vow:"sanitize=trim|lower,json"`
}

// Role is a closed set of membership levels.
//
//vow:enum sanitize=trim|lower, json
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)
