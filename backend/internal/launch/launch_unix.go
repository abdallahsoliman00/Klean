//go:build !windows

package launch

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/term"
)

// NeedsTerminal reports whether the program was started from a file manager
// (no terminal on stdin, stdout or stderr) inside a desktop session.
func NeedsTerminal() bool {
	if os.Getenv(envInTerminal) != "" {
		return false
	}
	for _, f := range []*os.File{os.Stdin, os.Stdout, os.Stderr} {
		if term.IsTerminal(int(f.Fd())) {
			return false
		}
	}
	if runtime.GOOS == "darwin" {
		return true
	}
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

// OwnsWindow reports whether this process runs in a window opened just for it.
func OwnsWindow() bool { return os.Getenv(envInTerminal) != "" }

// terminal is a terminal emulator and the arguments that come before the
// command it should run.
type terminal struct {
	name string
	args []string
}

// terminals lists emulators to try, the desktop's own first.
func terminals() []terminal {
	var list []terminal
	if t := os.Getenv("TERMINAL"); t != "" {
		list = append(list, terminal{t, []string{"-e"}})
	}
	list = append(list, terminal{"xdg-terminal-exec", nil})
	kde := []terminal{{"konsole", []string{"-e"}}}
	gnome := []terminal{{"ptyxis", []string{"--new-window", "--"}}, {"gnome-terminal", []string{"--"}}, {"kgx", []string{"-e"}}}
	desktop := strings.ToUpper(os.Getenv("XDG_CURRENT_DESKTOP"))
	if strings.Contains(desktop, "KDE") {
		list = append(list, kde...)
		list = append(list, gnome...)
	} else {
		list = append(list, gnome...)
		list = append(list, kde...)
	}
	return append(list,
		terminal{"x-terminal-emulator", []string{"-e"}},
		terminal{"xfce4-terminal", []string{"-x"}},
		terminal{"mate-terminal", []string{"-x"}},
		terminal{"alacritty", []string{"-e"}},
		terminal{"kitty", nil},
		terminal{"foot", nil},
		terminal{"wezterm", []string{"start", "--"}},
		terminal{"xterm", []string{"-e"}},
	)
}

// OpenInTerminal starts this program again inside a new terminal window. It
// returns false if no terminal could be started; the caller then simply keeps
// running without one.
func OpenInTerminal() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	env := append(os.Environ(), envInTerminal+"=1")

	if runtime.GOOS == "darwin" {
		// Terminal.app runs an executable file it is asked to open. (Arguments
		// can't be passed this way; a double-click has none anyway.)
		cmd := exec.Command("open", "-a", "Terminal", exe)
		cmd.Env = env
		return cmd.Run() == nil
	}

	for _, t := range terminals() {
		path, err := exec.LookPath(t.name)
		if err != nil {
			continue
		}
		args := append(append(append([]string{}, t.args...), exe), os.Args[1:]...)
		cmd := exec.Command(path, args...)
		cmd.Env = env
		if cmd.Start() == nil {
			go cmd.Wait()
			return true
		}
	}
	return false
}
