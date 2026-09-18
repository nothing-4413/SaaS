package audit

import (
	"time"

	"github.com/nothing-4413/saas/internal/platform/idgen"
)

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service { return &Service{store: store, now: time.Now} }
func (s *Service) Record(org, actor, action, resourceType, resourceID string, metadata map[string]interface{}) (Entry, error) {
	now := s.now().UTC()
	v := Entry{ID: idgen.New(), OrganizationID: org, ActorUserID: actor, Action: action, ResourceType: resourceType, ResourceID: resourceID, Metadata: metadata, CreatedAt: now}
	return v, s.store.Append(v)
}
func (s *Service) List(org string) []Entry { return s.store.List(org) }
