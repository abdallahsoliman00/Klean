//go:build !windows

package pdf

import (
	"os/exec"
	"syscall"
)

// ownTree starts cmd in a process group of its own, so killTree reaches the
// processes it starts and nothing else.
func ownTree(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

// killTree kills cmd and every process it started.
func killTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
