// Package launch handles how Klean starts on the desktop: giving it a
// terminal window when it was double-clicked, and opening the browser.
//
// The terminal window is the app's lifetime: closing it stops the server
// (SIGHUP on Linux and macOS, CTRL_CLOSE_EVENT on Windows), so the port is
// never left taken by a server nobody can see.
package launch

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// envInTerminal marks the copy of the program started inside a new terminal
// window, so it doesn't try to open another one.
const envInTerminal = "KLEAN_IN_TERMINAL"

// OpenBrowser opens url in the default browser without waiting for it.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait() // reap it
	return nil
}

// PauseIfOwnWindow keeps a window that exists only for this program open
// until Enter is pressed, so a startup error can be read before it closes.
func PauseIfOwnWindow() {
	if !OwnsWindow() {
		return
	}
	fmt.Fprint(os.Stderr, "\nPress Enter to close this window.")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}
