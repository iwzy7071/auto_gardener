package app

import "testing"

func TestNormalizePlanLimitsTreeCount(t *testing.T) {
	plan := GardenerPlan{}
	for i := 0; i < 12; i++ {
		plan.Trees = append(plan.Trees, TreePlan{Name: "task", Objective: "objective", Prompt: "prompt", Scope: []string{"scope"}})
	}

	got := normalizePlan(plan, &Task{MaxTreesPerForest: 5}, "instruction")
	if len(got.Trees) != 5 {
		t.Fatalf("tree count = %d, want 5", len(got.Trees))
	}
}
