package rbac

import "fmt"

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

func CanMutate(role Role) bool {
	return role == RoleAdmin || role == RoleMember
}

func CanAdminister(role Role) bool {
	return role == RoleAdmin
}

func Parse(role string) (Role, error) {
	switch Role(role) {
	case RoleAdmin, RoleMember, RoleViewer:
		return Role(role), nil
	default:
		return "", fmt.Errorf("unknown role: %s", role)
	}
}
