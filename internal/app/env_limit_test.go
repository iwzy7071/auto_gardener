package app

import "testing"

func TestGetenvIntMaxCapsConfiguredValue(t *testing.T) {
	t.Setenv("AUTO_GARDENER_MAX_CONCURRENT_TREES", "999")
	if got := getenvIntMax("AUTO_GARDENER_MAX_CONCURRENT_TREES", 3, maxConcurrentTreesLimit); got != maxConcurrentTreesLimit {
		t.Fatalf("getenvIntMax returned %d, want cap %d", got, maxConcurrentTreesLimit)
	}
}

func TestGetenvIntMaxKeepsSmallConfiguredValue(t *testing.T) {
	t.Setenv("AUTO_GARDENER_MAX_CONCURRENT_TREES", "4")
	if got := getenvIntMax("AUTO_GARDENER_MAX_CONCURRENT_TREES", 3, maxConcurrentTreesLimit); got != 4 {
		t.Fatalf("getenvIntMax returned %d, want 4", got)
	}
}

func TestNewOrchestratorCapsConcurrentTrees(t *testing.T) {
	t.Setenv("AUTO_GARDENER_MAX_CONCURRENT_TREES", "999")
	events := NewEventHub()
	store, err := NewStore(t.TempDir(), events)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(store, nil, store.DataDir(), "")
	if orch.maxConcurrent != maxConcurrentTreesLimit {
		t.Fatalf("maxConcurrent=%d, want %d", orch.maxConcurrent, maxConcurrentTreesLimit)
	}
}
