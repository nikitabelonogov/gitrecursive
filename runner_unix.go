//go:build unix

package main

import (
	"os/exec"
	"syscall"
)

// setupProcessGroup makes cancellation kill git's whole process tree
// (ssh, remote helpers, hooks), not just the git process itself.
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
