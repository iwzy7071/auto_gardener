package app

import (
	"fmt"
	"testing"
	"time"
)

func TestSummarizeUsageLimitsModelSummaries(t *testing.T) {
	records := make([]TokenUsageRecord, 0, maxUsageModelSummaries+5)
	for i := 0; i < maxUsageModelSummaries+5; i++ {
		records = append(records, TokenUsageRecord{
			ID:          fmt.Sprintf("record-%03d", i),
			TaskID:      "task",
			Model:       fmt.Sprintf("model-%03d", i),
			TotalTokens: int64(i + 1),
			CreatedAt:   time.Unix(int64(i), 0),
			ExactCost:   true,
		})
	}

	summary := summarizeUsage("task", records)
	if len(summary.Models) != maxUsageModelSummaries {
		t.Fatalf("expected %d model summaries, got %d", maxUsageModelSummaries, len(summary.Models))
	}
	if summary.Models[0].Model != "model-024" {
		t.Fatalf("expected highest-token model first, got %s", summary.Models[0].Model)
	}
	if summary.TotalTokens != 325 {
		t.Fatalf("expected total tokens to include all records, got %d", summary.TotalTokens)
	}
}
