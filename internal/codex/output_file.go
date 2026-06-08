package codex

import (
	"os"
	"path/filepath"
)

const privateOutputFileMode os.FileMode = 0600

func writeOutputFile(path string, data []byte) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, privateOutputFileMode); err != nil {
		return err
	}
	return os.Chmod(path, privateOutputFileMode)
}

func restrictOutputFile(path string) {
	if path == "" {
		return
	}
	_ = os.Chmod(path, privateOutputFileMode)
}
