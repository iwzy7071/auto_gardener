package app

import (
	"fmt"
	"testing"
)

func TestLimitUsageResponseRecordsKeepsRecentRecords(t *testing.T) {
	summary := TokenUsageSummary{Records: make([]TokenUsageRecord, 0, maxUsageResponseRecords+5)}
	for i := 0; i < maxUsageResponseRecords+5; i++ {
		summary.Records = append(summary.Records, TokenUsageRecord{ID: fmt.Sprintf("record-%03d", i)})
	}

	limited := limitUsageResponseRecords(summary)
	if len(limited.Records) != maxUsageResponseRecords {
		t.Fatalf("expected %d records, got %d", maxUsageResponseRecords, len(limited.Records))
	}
	if limited.Records[0].ID != "record-005" {
		t.Fatalf("expected oldest retained record-005, got %s", limited.Records[0].ID)
	}
}
