package webhook

import "time"

type Subscription struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	URL            string    `json:"url"`
	Secret         string    `json:"-"`
	EventTypes     []string  `json:"event_types"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateSubscriptionInput struct {
	URL        string   `json:"url"`
	Secret     string   `json:"secret"`
	EventTypes []string `json:"event_types"`
}
