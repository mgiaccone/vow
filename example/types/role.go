package types

// Role is a closed set of membership levels a workspace member can hold.
//
//vow:enum sanitize=trim|lower, json, sql, text
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)
