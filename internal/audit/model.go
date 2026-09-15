package audit

import "time"

type Entry struct {
	ID             string                 `json:"id"`
	OrganizationID string                 `json:"organization_id"`
	ActorUserID    string                 `json:"actor_user_id,omitempty"`
	Action         string                 `json:"action"`
	ResourceType   string                 `json:"resource_type"`
	ResourceID     string                 `json:"resource_id"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}
