package config

import "time"

// Reports printed to PDF by a browser installed on this machine.
const (
	// The page a report is printed on. The frontend's fallback (html2pdf.js)
	// uses the same size and margin.
	PDFPageSize = "A4"
	PDFMarginMM = 14 // on every side

	// PDFRenderTimeout is how long the browser may take before it is killed.
	PDFRenderTimeout = 30 * time.Second
	// PDFKillGrace is how long the browser's output is waited for once it was killed.
	PDFKillGrace = 2 * time.Second

	// PDFContentSecurityPolicy is put in front of a report before the browser
	// opens it. The model wrote the report, so it may only style itself and
	// show images and fonts it carries inline: no script, no frames, and
	// nothing loaded from the network or from files on this machine.
	PDFContentSecurityPolicy = "default-src 'none'; style-src 'unsafe-inline'; img-src data:; font-src data:; base-uri 'none'; form-action 'none'"

	// PDFDeadProxy is where the browser is told to send every request,
	// loopback ones included. Nothing listens there (port 9 is "discard"),
	// so no request leaves the browser.
	PDFDeadProxy = "http://127.0.0.1:9"

	// BrowserNone, given as --browser / KLEAN_BROWSER, turns PDF rendering off.
	BrowserNone = "none"
	// FlatpakProgram runs a browser installed as a Flatpak app.
	FlatpakProgram = "flatpak"
)
