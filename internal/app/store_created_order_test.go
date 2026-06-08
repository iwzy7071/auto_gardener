package app

import (
	"testing"
	"time"
)

func TestTreeCreatedOrderClampsOutOfRangeTimes(t *testing.T) {
	future := time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC)
	past := time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)

	if got := (&Tree{UpdatedAt: future}).CreatedOrder(); got != maxInt64 {
		t.Fatalf("future CreatedOrder = %d, want %d", got, maxInt64)
	}
	if got := (&Tree{UpdatedAt: past}).CreatedOrder(); got != minInt64 {
		t.Fatalf("past CreatedOrder = %d, want %d", got, minInt64)
	}
}
