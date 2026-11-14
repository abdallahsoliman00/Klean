package pdf

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"klean/backend/internal/config"
)

// What Prepare puts in front of a report. Coming first, the policy governs
// everything after it, and nothing in the report can end up before it. The
// page rules are !important so the report's own @page rules don't replace
// them, and print-color-adjust keeps backgrounds in print.
var preamble = fmt.Sprintf(`<!DOCTYPE html><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="%s">`+
	`<style>@page { size: %s !important; margin: %dmm !important } html { -webkit-print-color-adjust: exact; print-color-adjust: exact }</style>`+"\n",
	config.PDFContentSecurityPolicy, config.PDFPageSize, config.PDFMarginMM)

// httpEquiv finds the attribute that makes a <meta> act like a response
// header: a redirect ("refresh") is one the policy doesn't stop.
var httpEquiv = regexp.MustCompile(`(?i)(http)-(equiv)`)

// Prepare returns the report as the browser is given it: locked down by a
// Content-Security-Policy, set on the page config describes, and with its
// own <meta http-equiv> tags switched off. The hyphen becomes a character
// reference, which no longer names the attribute yet still reads as
// "http-equiv" where it was ordinary text.
func Prepare(html string) string {
	return preamble + httpEquiv.ReplaceAllString(html, "${1}&#45;${2}")
}

const (
	reportFile  = "report.html"
	pdfFile     = "report.pdf"
	profileDir  = "profile"
	tempPattern = "klean-pdf-"

	// The new profile starts with JavaScript blocked on every page. (The
	// switch that turns script off inside the renderer also breaks printing.)
	preferencesFile = "Default/Preferences"
	preferences     = `{"profile":{"default_content_setting_values":{"javascript":2}}}`
)

// command returns the program and arguments that print dir's report to a PDF
// next to it. The report isn't trusted even after Prepare, so the browser
// itself is also kept off the network (and its profile, which print writes,
// allows no script).
func command(b Browser, dir string) (string, []string) {
	args := []string{
		// The headless mode that is the whole browser: only it reads the profile's preferences.
		"--headless=new",
		// A profile of its own: the user's is neither read nor locked.
		"--user-data-dir=" + filepath.Join(dir, profileDir),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-extensions",
		"--disable-gpu",
		// The browser's own traffic (updates, metrics, sync).
		"--disable-background-networking",
		"--disable-component-update",
		// No name resolves, and every request (loopback too) goes to a proxy that isn't there.
		"--host-resolver-rules=MAP * ~NOTFOUND",
		"--proxy-server=" + config.PDFDeadProxy,
		"--proxy-bypass-list=<-loopback>",
		// No date, title, URL or page numbers around the page. (Older browsers know the second name.)
		"--no-pdf-header-footer",
		"--print-to-pdf-no-header",
		"--print-to-pdf=" + filepath.Join(dir, pdfFile),
		filepath.Join(dir, reportFile),
	}
	if !b.Flatpak {
		return b.Path, args
	}
	return config.FlatpakProgram, append([]string{"run", "--filesystem=" + dir, filepath.Base(b.Path)}, args...)
}

// Render prints the report to a PDF. It stops the browser when ctx ends or
// config.PDFRenderTimeout has passed.
func Render(ctx context.Context, b Browser, html string) ([]byte, error) {
	return print(ctx, b, Prepare(html))
}

// print has the browser print a document exactly as given.
func print(ctx context.Context, b Browser, document string) ([]byte, error) {
	dir, err := os.MkdirTemp("", tempPattern) // readable by this user only
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	prefs := filepath.Join(dir, profileDir, preferencesFile)
	if err := os.MkdirAll(filepath.Dir(prefs), 0o700); err != nil {
		return nil, err
	}
	for path, content := range map[string]string{prefs: preferences, filepath.Join(dir, reportFile): document} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return nil, err
		}
	}

	ctx, cancel := context.WithTimeout(ctx, config.PDFRenderTimeout)
	defer cancel()
	name, args := command(b, dir)
	cmd := exec.CommandContext(ctx, name, args...)
	ownTree(cmd)
	cmd.Cancel = func() error { return killTree(cmd) }
	cmd.WaitDelay = config.PDFKillGrace
	// Also once it has exited, so no helper process keeps the folder in use.
	defer killTree(cmd)

	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("%s was stopped: %w", name, ctx.Err())
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(output)))
	}
	pdf, err := os.ReadFile(filepath.Join(dir, pdfFile))
	if err != nil {
		return nil, fmt.Errorf("%s wrote no PDF: %s", name, strings.TrimSpace(string(output)))
	}
	return pdf, nil
}
