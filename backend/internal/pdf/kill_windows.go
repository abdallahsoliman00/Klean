//go:build windows

package pdf

import (
	"os/exec"
	"strconv"
)

// ownTree does nothing on Windows: taskkill follows the tree by itself.
func ownTree(cmd *exec.Cmd) {}

// killTree kills cmd and every process it started.
func killTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
}
