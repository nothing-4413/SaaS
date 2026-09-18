package alert

import "time"

type Rule struct {
	OrganizationID string    `json:"organization_id"`
	Threshold      int64     `json:"threshold"`
	Enabled        bool      `json:"enabled"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UpdateRuleInput struct {
	Threshold int64 `json:"threshold"`
	Enabled   bool  `json:"enabled"`
}
