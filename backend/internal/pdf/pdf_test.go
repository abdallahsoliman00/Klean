package pdf

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"klean/backend/internal/config"
)

func TestCandidates(t *testing.T) {
	env := map[string]string{"HOME": "/home/u", "ProgramFiles": `C:\Program Files`, "LocalAppData": `C:\Users\u\AppData\Local`}
	getenv := func(k string) string { return env[k] }
	for goos, want := range map[string][]string{
		"linux": {"google-chrome", "chromium", "microsoft-edge", "brave-browser",
			"/var/lib/flatpak/exports/bin/com.google.Chrome", "/home/u/.local/share/flatpak/exports/bin/com.brave.Browser"},
		"darwin": {"chromium", "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/home/u/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"},
		"windows": {"chrome", "msedge", `C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Users\u\AppData\Local\BraveSoftware\Brave-Browser\Application\brave.exe`},
	} {
		got := Candidates(goos, getenv)
		for _, w := range want {
			if !slices.Contains(got, w) {
				t.Errorf("%s: %q is missing from %q", goos, w, got)
			}
		}
	}
	// An unset variable adds no half-made paths.
	for _, c := range Candidates("windows", func(string) string { return "" }) {
		if strings.Contains(c, `\`) {
			t.Errorf("windows without environment: unexpected path %q", c)
		}
	}
	if got := Candidates("linux", getenv); got[0] != "google-chrome" || isFlatpak(got[0]) || !isFlatpak(got[len(got)-1]) {
		t.Errorf("linux: names on PATH should come before Flatpak apps: %q", got)
	}
}

func TestIsFlatpakAndSnap(t *testing.T) {
	if !isFlatpak("/var/lib/flatpak/exports/bin/com.google.Chrome") || isFlatpak("/usr/bin/google-chrome") {
		t.Error("isFlatpak should tell an exported launcher from a program")
	}
	if !isSnap("/usr/bin/snap") || isSnap("/usr/lib64/chromium-browser/chromium-browser") {
		t.Error("isSnap should tell the snap program from a browser")
	}
}

func TestFindNone(t *testing.T) {
	if Find(config.BrowserNone) != nil {
		t.Error("none should find no browser")
	}
}

func TestPrepare(t *testing.T) {
	got := Prepare(`<!doctype html><html><head><META HTTP-EQUIV="refresh" content="0;url=file:///etc/passwd"></head><body>about http-equiv</body></html>`)
	head := `<!DOCTYPE html><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="` + config.PDFContentSecurityPolicy + `">`
	if !strings.HasPrefix(got, head) {
		t.Errorf("the policy should come first: %s", got)
	}
	if !strings.Contains(got, "@page { size: A4 !important; margin: 14mm !important }") {
		t.Errorf("no page rule: %s", got)
	}
	if n := len(httpEquiv.FindAllString(got, -1)); n != 1 {
		t.Errorf("want only the policy's own http-equiv left, found %d: %s", n, got)
	}
	if !strings.Contains(got, `<META HTTP&#45;EQUIV="refresh"`) || !strings.Contains(got, "about http&#45;equiv</body>") {
		t.Errorf("http-equiv should be written with a character reference: %s", got)
	}
}

func TestCommand(t *testing.T) {
	name, args := command(Browser{Path: "/usr/bin/chromium"}, "/tmp/d")
	if name != "/usr/bin/chromium" || args[0] != "--headless=new" || args[len(args)-1] != "/tmp/d/report.html" {
		t.Errorf("unexpected command: %s %q", name, args)
	}
	for _, want := range []string{"--user-data-dir=/tmp/d/profile", "--print-to-pdf=/tmp/d/report.pdf", "--proxy-server=" + config.PDFDeadProxy} {
		if !slices.Contains(args, want) {
			t.Errorf("%s is missing from %q", want, args)
		}
	}
	name, flatpak := command(Browser{Path: "/var/lib/flatpak/exports/bin/com.google.Chrome", Flatpak: true}, "/tmp/d")
	if name != "flatpak" || !slices.Equal(flatpak, append([]string{"run", "--filesystem=/tmp/d", "com.google.Chrome"}, args...)) {
		t.Errorf("unexpected Flatpak command: %s %q", name, flatpak)
	}
}

// ---------------- with a real browser ----------------

// realBrowser returns the browser installed here (or the one KLEAN_BROWSER
// names), skipping the test when there is none.
func realBrowser(t *testing.T) Browser {
	t.Helper()
	b := Find(os.Getenv("KLEAN_BROWSER"))
	if b == nil {
		t.Skip("no Chromium-based browser is installed")
	}
	return *b
}

// text returns the text a PDF reader extracts, skipping the test without pdftotext.
func text(t *testing.T, pdf []byte) string {
	t.Helper()
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatalf("not a PDF: %.40q", pdf)
	}
	cmd := exec.Command("pdftotext", "-", "-")
	cmd.Stdin = bytes.NewReader(pdf)
	out, err := cmd.Output()
	if errors.Is(err, exec.ErrNotFound) {
		t.Skip("pdftotext (poppler) is not installed, so the PDF's text can't be read")
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestRenderReport(t *testing.T) {
	b := realBrowser(t)
	// The report's own page rule must not win over ours.
	pdf, err := Render(context.Background(), b, `<!doctype html><html><head><title>Q3</title>
<style>@page { size: A3 landscape; margin: 0 } h1 { background: #246; color: #fff }</style></head>
<body><h1>Quarterly figures</h1><p>Revenue rose by 12 percent.</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	// A4 is 595 by 842 points.
	if !bytes.Contains(pdf, []byte("/MediaBox [0 0 594.9")) && !bytes.Contains(pdf, []byte("/MediaBox [0 0 595")) {
		t.Error("the page is not A4 portrait")
	}
	got := text(t, pdf)
	for _, want := range []string{"Quarterly figures", "Revenue rose by 12 percent."} {
		if !strings.Contains(got, want) {
			t.Errorf("%q can't be extracted as text: %q", want, got)
		}
	}
	// No header or footer: nothing but the report's own two lines.
	if len(strings.Fields(got)) != 7 {
		t.Errorf("unexpected text beside the report's own: %q", got)
	}
}

// recorder is a local server that counts the requests it gets.
func recorder(t *testing.T) (url string, hits *atomic.Int32) {
	t.Helper()
	hits = new(atomic.Int32)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("LOADED"))
	}))
	t.Cleanup(srv.Close)
	return srv.URL, hits
}

func TestRenderHostileReport(t *testing.T) {
	b := realBrowser(t)
	url, hits := recorder(t)
	// /etc/passwd is readable even inside a Flatpak sandbox; elsewhere any file URL shows the same.
	const local = "file:///etc/passwd"
	doc := strings.NewReplacer("URL", url, "LOCAL", local).Replace(`<!doctype html><html><head>
<meta http-equiv="refresh" content="0;url=LOCAL">
<noscript><meta http-equiv="refresh" content="0;url=URL/refresh"></noscript>
<base href="URL/base/">
<link rel="stylesheet" href="URL/style.css">
<link rel="preconnect" href="URL">
<style>@import url("URL/import.css"); body { background: url("URL/bg.png") } @font-face { font-family: f; src: url("URL/f.woff2") } p { font-family: f }</style>
</head><body onload="document.body.append('SCRIPTRAN-ONLOAD')">
<p>Still the report</p>
<script>document.write("SCRIPTRAN-INLINE")</script>
<script src="URL/script.js"></script>
<img src="URL/img.png" onerror="document.body.append('SCRIPTRAN-ONERROR')">
<img src="relative.png"><img src="LOCAL">
<iframe src="LOCAL"></iframe><iframe src="URL/frame"></iframe><iframe srcdoc="<img src='URL/srcdoc.png'><script>document.write('SCRIPTRAN-SRCDOC')</script>"></iframe>
<object data="LOCAL"></object><embed src="LOCAL">
<svg><image href="URL/svg.png"/></svg>
<a href="javascript:document.body.append('SCRIPTRAN-LINK')">link</a>
</body></html>`)
	pdf, err := Render(context.Background(), b, doc)
	if err != nil {
		t.Fatal(err)
	}
	got := text(t, pdf)
	if !strings.Contains(got, "Still the report") {
		t.Errorf("the report is gone, so the browser went elsewhere: %q", got)
	}
	for _, bad := range []string{"SCRIPTRAN", "root:", "LOADED"} {
		if strings.Contains(got, bad) {
			t.Errorf("%q got into the PDF: %q", bad, got)
		}
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("the browser made %d requests", n)
	}
}

// The browser's own switches must hold even without the policy Prepare adds.
func TestBrowserSwitchesAlone(t *testing.T) {
	b := realBrowser(t)
	url, hits := recorder(t)
	doc := strings.ReplaceAll(`<!doctype html><html><head><link rel="stylesheet" href="URL/style.css"></head><body>
<p>Still the report</p><script>document.write("SCRIPTRAN")</script>
<img src="URL/img.png"><img src="http://klean-test.invalid/img.png"><iframe src="URL/frame"></iframe></body></html>`, "URL", url)
	pdf, err := print(context.Background(), b, doc)
	if err != nil {
		t.Fatal(err)
	}
	got := text(t, pdf)
	if !strings.Contains(got, "Still the report") || strings.Contains(got, "SCRIPTRAN") || strings.Contains(got, "LOADED") {
		t.Errorf("script ran or something was loaded: %q", got)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("the browser made %d requests", n)
	}
}

func TestRenderStopsWithContext(t *testing.T) {
	b := realBrowser(t)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := Render(ctx, b, "<!doctype html><p>too slow</p>")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("want the context's error, got %v", err)
	}
	if took := time.Since(start); took > 200*time.Millisecond+2*config.PDFKillGrace {
		t.Errorf("stopping the browser took %s", took)
	}
}
