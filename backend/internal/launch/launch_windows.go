//go:build windows

package launch

import (
	"syscall"
	"unsafe"
)

var getConsoleProcessList = syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleProcessList")

// NeedsTerminal is always false on Windows: a console program started from
// Explorer gets its own console window automatically, and closing that window
// ends the program.
func NeedsTerminal() bool { return false }

// OpenInTerminal is never needed on Windows; see NeedsTerminal.
func OpenInTerminal() bool { return false }

// OwnsWindow reports whether the console window was created for this process
// alone, which is the case when it was double-clicked in Explorer.
func OwnsWindow() bool {
	var pids [2]uint32
	n, _, _ := getConsoleProcessList.Call(uintptr(unsafe.Pointer(&pids[0])), 2)
	return n == 1
}
