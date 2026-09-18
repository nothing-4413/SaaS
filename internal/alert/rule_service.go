package alert

import (
	"strings"
	"time"
)

type RuleService struct {
	store Store
	now   func() time.Time
}

func NewRuleService(store Store) *RuleService { return &RuleService{store: store, now: time.Now} }

func (s *RuleService) Update(org string, input UpdateRuleInput) (Rule, error) {
	if strings.TrimSpace(org) == "" || input.Threshold < 0 {
		return Rule{}, ErrInvalidInput
	}
	value := Rule{OrganizationID: org, Threshold: input.Threshold, Enabled: input.Enabled, UpdatedAt: s.now().UTC()}
	return value, s.store.Put(value)
}

func (s *RuleService) Get(org string) (Rule, error) { return s.store.Get(org) }
func (s *RuleService) ListEnabled() []Rule          { return s.store.ListEnabled() }
