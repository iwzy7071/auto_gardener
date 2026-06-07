package app

import "testing"

func TestParseNumberRejectsInvalidOrHugeUsageCounts(t *testing.T) {
	for _, in := range []string{"", "999999999999999999999999", "1000000000001", "1e309"} {
		if n, ok := parseNumber(in); ok {
			t.Fatalf("parseNumber(%q) = %d, true; want false", in, n)
		}
	}
}

func TestParseTokenDetailLineRejectsHugeUsageCounts(t *testing.T) {
	if _, _, ok := parseTokenDetailLine("total tokens: 1000000000001"); ok {
		t.Fatal("parseTokenDetailLine accepted over-limit token count")
	}
	kind, n, ok := parseTokenDetailLine("total tokens: 1,000")
	if !ok || kind != "total" || n != 1000 {
		t.Fatalf("parseTokenDetailLine valid count = %q %d %v, want total 1000 true", kind, n, ok)
	}
}
