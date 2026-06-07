package app

import (
	"strings"
	"testing"
)

func TestNormalizePlanLimitsModelControlledText(t *testing.T) {
	scope := make([]string, maxPlanScopeItems+5)
	for i := range scope {
		scope[i] = strings.Repeat("s", maxPlanScopeRunes+10)
	}
	plan := GardenerPlan{
		MessageToUser: strings.Repeat("m", maxPlanMessageRunes+10),
		Trees: []TreePlan{{
			Name:      strings.Repeat("n", maxPlanNameRunes+10),
			Objective: strings.Repeat("o", maxPlanObjectiveRunes+10),
			Prompt:    strings.Repeat("p", maxPlanPromptRunes+10),
			Scope:     scope,
		}},
	}

	got := normalizePlan(plan, nil, "instruction")
	if len([]rune(got.MessageToUser)) != maxPlanMessageRunes {
		t.Fatalf("message length = %d, want %d", len([]rune(got.MessageToUser)), maxPlanMessageRunes)
	}
	tr := got.Trees[0]
	if len([]rune(tr.Name)) != maxPlanNameRunes {
		t.Fatalf("name length = %d, want %d", len([]rune(tr.Name)), maxPlanNameRunes)
	}
	if len([]rune(tr.Objective)) != maxPlanObjectiveRunes {
		t.Fatalf("objective length = %d, want %d", len([]rune(tr.Objective)), maxPlanObjectiveRunes)
	}
	if len([]rune(tr.Prompt)) != maxPlanPromptRunes {
		t.Fatalf("prompt length = %d, want %d", len([]rune(tr.Prompt)), maxPlanPromptRunes)
	}
	if len(tr.Scope) != maxPlanScopeItems {
		t.Fatalf("scope item count = %d, want %d", len(tr.Scope), maxPlanScopeItems)
	}
	for _, item := range tr.Scope {
		if len([]rune(item)) != maxPlanScopeRunes {
			t.Fatalf("scope length = %d, want %d", len([]rune(item)), maxPlanScopeRunes)
		}
	}
}
