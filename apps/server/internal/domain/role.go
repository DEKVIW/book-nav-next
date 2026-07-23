package domain

// Role 用户角色，替代旧版 is_admin + is_superadmin 双布尔。
type Role string

const (
	RoleUser       Role = "user"
	RoleAdmin      Role = "admin"
	RoleSuperAdmin Role = "superadmin"
)

func (r Role) IsAdmin() bool {
	return r == RoleAdmin || r == RoleSuperAdmin
}

func (r Role) IsSuperAdmin() bool {
	return r == RoleSuperAdmin
}

func (r Role) Valid() bool {
	switch r {
	case RoleUser, RoleAdmin, RoleSuperAdmin:
		return true
	default:
		return false
	}
}
