package audit

import (
	"fmt"
	"sync/atomic"
	"time"
)

type Service struct {
	store Store
	now   func() time.Time
	seq   uint64
}

func NewService(store Store) *Service { return &Service{store: store, now: time.Now} }
func (s *Service) Record(org, actor, action, resourceType, resourceID string, metadata map[string]interface{}) (Entry, error) {
	now := s.now().UTC()
	v := Entry{ID: fmt.Sprintf("%d-%d", now.UnixNano(), atomic.AddUint64(&s.seq, 1)), OrganizationID: org, ActorUserID: actor, Action: action, ResourceType: resourceType, ResourceID: resourceID, Metadata: metadata, CreatedAt: now}
	return v, s.store.Append(v)
}
func (s *Service) List(org string) []Entry { return s.store.List(org) }
