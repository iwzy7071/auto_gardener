package app

import "testing"

func TestEventHubLimitsSubscribersPerTask(t *testing.T) {
	hub := NewEventHub()
	unsubscribers := make([]func(), 0, maxEventSubscribersPerTask)
	for i := 0; i < maxEventSubscribersPerTask; i++ {
		_, unsubscribe, ok := hub.Subscribe("task1")
		if !ok {
			t.Fatalf("subscriber %d unexpectedly rejected", i)
		}
		unsubscribers = append(unsubscribers, unsubscribe)
	}
	_, unsubscribe, ok := hub.Subscribe("task1")
	defer unsubscribe()
	if ok {
		t.Fatal("subscriber over limit accepted")
	}
	unsubscribers[0]()
	_, unsubscribe, ok = hub.Subscribe("task1")
	defer unsubscribe()
	if !ok {
		t.Fatal("subscriber rejected after a slot was freed")
	}
	for _, unsubscribe := range unsubscribers[1:] {
		unsubscribe()
	}
}
