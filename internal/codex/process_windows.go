//go:build windows

package codex

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

func setProcessGroup(cmd *exec.Cmd) {
	// Windows does not support Unix process groups. Cancellation is handled
	// by taskkill /T so Codex child processes are terminated together.
}

const taskkillTimeout = 5 * time.Second

func killProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), taskkillTimeout)
	defer cancel()
	if err := exec.CommandContext(ctx, "taskkill", "/T", "/F", "/PID", fmt.Sprint(cmd.Process.Pid)).Run(); err == nil {
		return nil
	}
	return cmd.Process.Kill()
}
