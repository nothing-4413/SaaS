package auth

import "time"

type Permission string

const (
	PermissionUserRead        Permission = "user:read"
	PermissionUserWrite       Permission = "user:write"
	PermissionRoleRead        Permission = "role:read"
	PermissionRoleManage      Permission = "role:manage"
	PermissionProductRead     Permission = "product:read"
	PermissionProductWrite    Permission = "product:write"
	PermissionProductManage   Permission = "product:manage"
	PermissionInventoryRead   Permission = "inventory:read"
	PermissionInventoryWrite  Permission = "inventory:write"
	PermissionInventoryImport Permission = "inventory:import"
	PermissionInventoryExport Permission = "inventory:export"
	PermissionInventoryManage Permission = "inventory:manage"
	PermissionOrderRead       Permission = "order:read"
	PermissionOrderWrite      Permission = "order:write"
	PermissionOrderApprove    Permission = "order:approve"
	PermissionOrderManage     Permission = "order:manage"
	PermissionReportRead      Permission = "report:read"
	PermissionReportExport    Permission = "report:export"
	PermissionAuditRead       Permission = "audit:read"
	PermissionWebhookManage   Permission = "webhook:manage"
)

var AllPermissions = []Permission{
	PermissionUserRead,
	PermissionUserWrite,
	PermissionRoleRead,
	PermissionRoleManage,
	PermissionProductRead,
	PermissionProductWrite,
	PermissionProductManage,
	PermissionInventoryRead,
	PermissionInventoryWrite,
	PermissionInventoryImport,
	PermissionInventoryExport,
	PermissionInventoryManage,
	PermissionOrderRead,
	PermissionOrderWrite,
	PermissionOrderApprove,
	PermissionOrderManage,
	PermissionReportRead,
	PermissionReportExport,
	PermissionAuditRead,
	PermissionWebhookManage,
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
	Active         bool      `json:"active"`
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

type UpdateUserInput struct {
	Name     string   `json:"name"`
	Password string   `json:"password"`
	RoleIDs  []string `json:"role_ids"`
	Active   *bool    `json:"active,omitempty"`
}

type Session struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	UserID         string     `json:"user_id"`
	ExpiresAt      time.Time  `json:"expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}

type RefreshInput struct {
	RefreshToken string `json:"refresh_token"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type PasswordResetRequestInput struct {
	Email string `json:"email"`
}

type PasswordResetConfirmInput struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type CreateRoleInput struct {
	Name        string       `json:"name"`
	Permissions []Permission `json:"permissions"`
}

type UpdateRoleInput struct {
	Name        string       `json:"name"`
	Permissions []Permission `json:"permissions"`
}
