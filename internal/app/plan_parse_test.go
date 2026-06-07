package app

import (
	"strings"
	"testing"
)

func TestParsePlanRejectsOversizedJSON(t *testing.T) {
	_, err := parsePlan(`{"forest_finished":false,"trees":[{"name":"` + strings.Repeat("x", maxPlanJSONBytes) + `"}]}`)
	if err == nil || !strings.Contains(err.Error(), "JSON 对象过大") {
		t.Fatalf("parsePlan error = %v, want oversized JSON error", err)
	}
}

func TestParsePlanAcceptsSmallJSON(t *testing.T) {
	plan, err := parsePlan(`prefix {"forest_finished":true,"trees":[]} suffix`)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.ForestFinished || len(plan.Trees) != 0 {
		t.Fatalf("plan = %+v", plan)
	}
}
