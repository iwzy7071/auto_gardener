package app

import "testing"

func TestParsePowerCfgIndexes(t *testing.T) {
	ac, dc := parsePowerCfgIndexes(`
    Current AC Power Setting Index: 0x00000000
    Current DC Power Setting Index: 0x00000384
`)
	if ac != 0 || dc != 900 {
		t.Fatalf("got ac=%d dc=%d", ac, dc)
	}
}

func TestParsePMSetValues(t *testing.T) {
	vals := parsePMSetValues(`Battery Power:
 sleep                10
 standby              1
AC Power:
 sleep                0
 autopoweroff         0
`)
	if vals["Battery Power"]["sleep"] != 10 || vals["Battery Power"]["standby"] != 1 {
		t.Fatalf("bad battery values: %#v", vals["Battery Power"])
	}
	if vals["AC Power"]["sleep"] != 0 || vals["AC Power"]["autopoweroff"] != 0 {
		t.Fatalf("bad ac values: %#v", vals["AC Power"])
	}
}

func TestLimitedPowerOutputCapsBufferedBytes(t *testing.T) {
	var out limitedPowerOutput
	data := make([]byte, maxPowerCommandOutputBytes+1)
	n, err := out.Write(data)
	if n != len(data) {
		t.Fatalf("Write n=%d, want %d", n, len(data))
	}
	if err == nil {
		t.Fatalf("expected oversized power output write to fail")
	}
	if len(out.bytes) != maxPowerCommandOutputBytes {
		t.Fatalf("buffered bytes=%d, want %d", len(out.bytes), maxPowerCommandOutputBytes)
	}
	if !out.truncated {
		t.Fatalf("expected output to be marked truncated")
	}
}
