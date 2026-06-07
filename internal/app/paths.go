package app

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func DefaultDataDir() string {
	if v := os.Getenv("AUTO_GARDENER_DATA"); strings.TrimSpace(v) != "" {
		return expandHome(v)
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "forest_data"
	}
	desktop := filepath.Join(home, "Desktop")
	if st, err := os.Stat(desktop); err == nil && st.IsDir() {
		return filepath.Join(desktop, "forest_data")
	}
	return filepath.Join(home, "forest_data")
}

func expandHome(p string) string {
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~"+string(os.PathSeparator)) {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

var unsafePathChars = regexp.MustCompile(`[^\p{Han}\p{L}\p{N}._-]+`)

func safeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "untitled"
	}
	s = unsafePathChars.ReplaceAllString(s, "_")
	s = strings.Trim(s, "._- ")
	if s == "" {
		return "untitled"
	}
	r := []rune(s)
	if len(r) > 36 {
		s = string(r[:36])
	}
	return s
}
