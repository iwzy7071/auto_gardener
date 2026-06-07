package app

import (
	"strings"
	"testing"
)

func TestModelPricesRejectsOversizedEnv(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{"custom":{"inputPerMTok":99}`)
	for i := 0; b.Len() <= maxModelPricesJSONBytes; i++ {
		b.WriteString(`,"dummy`)
		b.WriteString(strings.Repeat("x", 32))
		b.WriteString(`":{"inputPerMTok":1}`)
	}
	b.WriteString(`}`)
	t.Setenv("AUTO_GARDENER_MODEL_PRICES_JSON", b.String())
	prices := modelPrices()
	if _, ok := prices["custom"]; ok {
		t.Fatalf("oversized model price env should be ignored: %#v", prices["custom"])
	}
}

func TestModelPricesAcceptsBoundedEnv(t *testing.T) {
	t.Setenv("AUTO_GARDENER_MODEL_PRICES_JSON", `{"custom":{"inputPerMTok":1,"cachedInputPerMTok":0.5,"outputPerMTok":2}}`)
	prices := modelPrices()
	if prices["custom"].InputPerMTok != 1 || prices["custom"].OutputPerMTok != 2 {
		t.Fatalf("bounded model price env not applied: %#v", prices["custom"])
	}
}
