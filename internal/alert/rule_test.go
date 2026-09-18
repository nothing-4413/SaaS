package alert

import "testing"

func TestRuleServiceUpdatesAndListsEnabledRules(t *testing.T) {
	service := NewRuleService(NewMemoryStore())
	value, err := service.Update("org", UpdateRuleInput{Threshold: 5, Enabled: true})
	if err != nil || value.Threshold != 5 {
		t.Fatalf("value=%+v err=%v", value, err)
	}
	if enabled := service.ListEnabled(); len(enabled) != 1 || enabled[0].OrganizationID != "org" {
		t.Fatalf("enabled=%+v", enabled)
	}
	if _, err := service.Update("org", UpdateRuleInput{Threshold: -1, Enabled: true}); err != ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
}
