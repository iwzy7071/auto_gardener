//go:build windows

package codex

import (
	"fmt"
	"os/exec"
)

func setProcessGroup(cmd *exec.Cmd) {
	// Windows does not support Unix process groups. Cancellation is handled
	// by taskkill /T so Codex child processes are terminated together.
}

func killProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if err := exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprint(cmd.Process.Pid)).Run(); err == nil {
		return nil
	}
	return cmd.Process.Kill()
}
