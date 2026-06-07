package app

import "testing"

func TestIsHiddenOrNoiseFileHidesEnvFiles(t *testing.T) {
	for _, path := range []string{".env", "config/.env"} {
		if !isHiddenOrNoiseFile(path) {
			t.Fatalf("expected %s to be hidden", path)
		}
	}
}
