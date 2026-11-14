// Package pdf prints a report (an HTML document the model wrote) to a PDF
// with real text, using a Chromium-based browser already installed on this
// machine, run headless.
package pdf

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"klean/backend/internal/config"
)

// Browser is a Chromium-based browser that can print to PDF.
type Browser struct {
	// Path is the executable. For a Flatpak app it is the launcher Flatpak
	// exports, whose file name is the app's ID.
	Path string
	// Flatpak apps are sandboxed away from the temporary folder, so they
	// are started through "flatpak run" with that folder shared.
	Flatpak bool
}

// Chrome, Chromium, Edge and Brave, in the order they are tried.
var (
	// Names looked up on PATH.
	unixNames    = []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "microsoft-edge", "microsoft-edge-stable", "brave-browser", "brave"}
	windowsNames = []string{"chrome", "chromium", "msedge", "brave"}

	macApps     = []string{"Google Chrome", "Chromium", "Microsoft Edge", "Brave Browser"}
	flatpakApps = []string{"com.google.Chrome", "org.chromium.Chromium", "com.microsoft.Edge", "com.brave.Browser"}
	// Below Program Files and the user's local application data.
	windowsApps = []string{
		`Google\Chrome\Application\chrome.exe`,
		`Chromium\Application\chrome.exe`,
		`Microsoft\Edge\Application\msedge.exe`,
		`BraveSoftware\Brave-Browser\Application\brave.exe`,
	}
	windowsRoots = []string{"ProgramFiles", "ProgramFiles(x86)", "LocalAppData"}
)

const (
	flatpakSystemExports = "/var/lib/flatpak/exports/bin"
	flatpakUserExports   = ".local/share/flatpak/exports/bin" // below the home folder
)

// Candidates lists where a browser may be on the given OS, best first: names
// to look up on PATH, then absolute paths. getenv reads an environment
// variable ("" when unset).
func Candidates(goos string, getenv func(string) string) []string {
	home := getenv("HOME")
	switch goos {
	case "windows":
		list := append([]string{}, windowsNames...)
		for _, root := range windowsRoots {
			if dir := getenv(root); dir != "" {
				for _, app := range windowsApps {
					list = append(list, dir+`\`+app)
				}
			}
		}
		return list
	case "darwin":
		list := append([]string{}, unixNames...)
		for _, root := range []string{"", home} {
			for _, app := range macApps {
				list = append(list, root+"/Applications/"+app+".app/Contents/MacOS/"+app)
			}
		}
		return list
	default:
		list := append([]string{}, unixNames...)
		exports := []string{flatpakSystemExports}
		if home != "" {
			exports = append(exports, home+"/"+flatpakUserExports)
		}
		for _, dir := range exports {
			for _, app := range flatpakApps {
				list = append(list, dir+"/"+app)
			}
		}
		return list
	}
}

// isFlatpak reports whether path is a launcher exported by Flatpak.
func isFlatpak(path string) bool {
	return strings.HasSuffix(filepath.ToSlash(filepath.Dir(path)), "/flatpak/exports/bin")
}

// isSnap reports whether resolved, a path with its symlinks followed, is a
// Snap app: those are all links to the snap program. Snap browsers can't
// read the temporary folder the report is put in.
func isSnap(resolved string) bool { return filepath.Base(resolved) == "snap" }

// Find returns the browser to print with, or nil when there is none.
// override is --browser / KLEAN_BROWSER: "" looks for an installed one,
// config.BrowserNone means none, anything else is the only program tried.
func Find(override string) *Browser {
	candidates := []string{override}
	switch override {
	case config.BrowserNone:
		return nil
	case "":
		candidates = Candidates(runtime.GOOS, os.Getenv)
	}
	for _, c := range candidates {
		path, err := exec.LookPath(c)
		if err != nil {
			continue
		}
		if resolved, err := filepath.EvalSymlinks(path); err == nil && isSnap(resolved) {
			continue
		}
		return &Browser{Path: path, Flatpak: isFlatpak(path)}
	}
	return nil
}
