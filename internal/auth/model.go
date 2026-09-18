package auth

import "time"

type Permission string

const (
	PermissionUserRead        Permission = "user:read"
	PermissionUserWrite       Permission = "user:write"
	PermissionRoleManage      Permission = "role:manage"
	PermissionProductManage   Permission = "product:manage"
	PermissionInventoryManage Permission = "inventory:manage"
	PermissionOrderManage     Permission = "order:manage"
	PermissionReportRead      Permission = "report:read"
	PermissionAuditRead       Permission = "audit:read"
)

var AllPermissions = []Permission{
	PermissionUserRead,
	PermissionUserWrite,
	PermissionRoleManage,
	PermissionProductManage,
	PermissionInventoryManage,
	PermissionOrderManage,
	PermissionReportRead,
	PermissionAuditRead,
}

type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Role struct {
	ID             string       `json:"id"`
	OrganizationID string       `json:"organization_id"`
	Name           string       `json:"name"`
	Permissions    []Permission `json:"permissions"`
}

type User struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	RoleIDs        []string  `json:"role_ids"`
	PasswordHash   string    `json:"-"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateOrganizationInput struct {
	Name          string `json:"name"`
	OwnerEmail    string `json:"owner_email"`
	OwnerName     string `json:"owner_name"`
	OwnerPassword string `json:"owner_password"`
}

type CreateUserInput struct {
	Email    string   `json:"email"`
	Name     string   `json:"name"`
	Password string   `json:"password"`
	RoleIDs  []string `json:"role_ids"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateRoleInput struct {
	Name        string       `json:"name"`
	Permissions []Permission `json:"permissions"`
}
