package auth

import "time"

type Permission string

const (
	PermissionUserRead   Permission = "user:read"
	PermissionUserWrite  Permission = "user:write"
	PermissionRoleManage Permission = "role:manage"
)

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
	CreatedAt      time.Time `json:"created_at"`
}

type CreateOrganizationInput struct {
	Name string `json:"name"`
}

type CreateUserInput struct {
	Email   string   `json:"email"`
	Name    string   `json:"name"`
	RoleIDs []string `json:"role_ids"`
}

type CreateRoleInput struct {
	Name        string       `json:"name"`
	Permissions []Permission `json:"permissions"`
}
