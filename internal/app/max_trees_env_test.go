package app

import "testing"

func TestGetenvIntFallbackMaxCapsPrimaryValue(t *testing.T) {
	t.Setenv("AUTO_GARDENER_MAX_TREES_PER_FOREST", "999")
	if got := getenvIntFallbackMax("AUTO_GARDENER_MAX_TREES_PER_FOREST", "AUTO_GARDENER_MAX_TREES_PER_WAVE", 5, maxTreesPerForestLimit); got != maxTreesPerForestLimit {
		t.Fatalf("getenvIntFallbackMax returned %d, want cap %d", got, maxTreesPerForestLimit)
	}
}

func TestGetenvIntFallbackMaxCapsFallbackValue(t *testing.T) {
	t.Setenv("AUTO_GARDENER_MAX_TREES_PER_WAVE", "999")
	if got := getenvIntFallbackMax("AUTO_GARDENER_MAX_TREES_PER_FOREST", "AUTO_GARDENER_MAX_TREES_PER_WAVE", 5, maxTreesPerForestLimit); got != maxTreesPerForestLimit {
		t.Fatalf("getenvIntFallbackMax returned %d, want cap %d", got, maxTreesPerForestLimit)
	}
}

func TestNewOrchestratorCapsMaxTrees(t *testing.T) {
	t.Setenv("AUTO_GARDENER_MAX_TREES_PER_FOREST", "999")
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(store, nil, store.DataDir(), "")
	if orch.maxTrees != maxTreesPerForestLimit {
		t.Fatalf("maxTrees=%d, want %d", orch.maxTrees, maxTreesPerForestLimit)
	}
}
