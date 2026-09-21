package webhook

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nothing-4413/saas/internal/outbox"
	"github.com/nothing-4413/saas/internal/platform/idgen"
)

type SubscriptionService struct {
	store  Store
	sender Sender
	now    func() time.Time
}

func NewSubscriptionService(store Store, sender Sender) *SubscriptionService {
	return &SubscriptionService{store: store, sender: sender, now: time.Now}
}

func (s *SubscriptionService) Create(org string, input CreateSubscriptionInput) (Subscription, error) {
	endpoint := strings.TrimSpace(input.URL)
	parsed, err := url.ParseRequestURI(endpoint)
	if strings.TrimSpace(org) == "" || err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || len(input.Secret) < 16 || len(input.EventTypes) == 0 {
		return Subscription{}, ErrInvalidInput
	}
	events := make([]string, 0, len(input.EventTypes))
	seen := make(map[string]struct{}, len(input.EventTypes))
	for _, eventType := range input.EventTypes {
		eventType = strings.TrimSpace(eventType)
		if eventType == "" {
			return Subscription{}, ErrInvalidInput
		}
		if _, ok := seen[eventType]; !ok {
			seen[eventType] = struct{}{}
			events = append(events, eventType)
		}
	}
	value := Subscription{ID: idgen.New(), OrganizationID: org, URL: endpoint, Secret: input.Secret, EventTypes: events, Active: true, CreatedAt: s.now().UTC()}
	return value, s.store.Create(value)
}

func (s *SubscriptionService) List(org string) []Subscription { return s.store.List(org) }

func (s *SubscriptionService) Delete(org, id string) error {
	if strings.TrimSpace(org) == "" || strings.TrimSpace(id) == "" {
		return ErrInvalidInput
	}
	return s.store.Delete(org, id)
}

func (s *SubscriptionService) Deliver(event outbox.Event) error {
	var failures []error
	for _, subscription := range s.store.List(event.OrganizationID) {
		if !subscription.Active || !accepts(subscription.EventTypes, event.Type) {
			continue
		}
		if s.store.IsDelivered(event.ID, subscription.ID) {
			continue
		}
		err := s.sender.Send(Delivery{
			ID:             event.ID + ":" + subscription.ID,
			URL:            subscription.URL,
			Secret:         subscription.Secret,
			EventType:      event.Type,
			Payload:        event.Payload,
			IdempotencyKey: event.ID + ":" + subscription.ID,
		})
		if err != nil {
			failures = append(failures, fmt.Errorf("subscription %s: %w", subscription.ID, err))
			continue
		}
		if err := s.store.MarkDelivered(event.ID, subscription.ID, s.now().UTC()); err != nil {
			failures = append(failures, fmt.Errorf("record subscription %s delivery: %w", subscription.ID, err))
		}
	}
	return errors.Join(failures...)
}

func accepts(events []string, eventType string) bool {
	for _, candidate := range events {
		// Password reset payloads contain a one-time secret. A wildcard
		// subscription must never receive them accidentally; opt in explicitly.
		if candidate == eventType || (candidate == "*" && eventType != "auth.password_reset_requested") {
			return true
		}
	}
	return false
}

func statusForError(err error) int {
	if errors.Is(err, ErrNotFound) {
		return 404
	}
	if errors.Is(err, ErrConflict) {
		return 409
	}
	return 400
}
