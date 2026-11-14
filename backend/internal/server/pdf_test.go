package server_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"klean/backend/internal/pdf"
	"klean/backend/internal/server"
	"klean/backend/internal/store"
	"klean/backend/internal/tools"
)

// pdfSetup returns a server printing with browser (nil for none) and the
// IDs of a report and of an artifact that isn't one.
func pdfSetup(t *testing.T, browser *pdf.Browser) (api *httptest.Server, report, page string) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	chat, err := db.CreateChat(store.NewChat{})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, kind := range []string{tools.KindReport, "html"} {
		a, err := db.CreateArtifact(chat.ID, "Figures", kind, "", "<!doctype html><html><body><h1>Quarterly figures</h1></body></html>")
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, a.ID)
	}
	api = httptest.NewServer(server.New(db, nil, "*", server.WithBrowser(browser)))
	t.Cleanup(func() { api.Close(); db.Close() })
	return api, ids[0], ids[1]
}

// postPDF asks for an artifact's PDF.
func postPDF(t *testing.T, api *httptest.Server, id string) (status int, contentType string, body []byte) {
	t.Helper()
	res, err := http.Post(api.URL+"/api/artifacts/"+id+"/pdf", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ = io.ReadAll(res.Body)
	return res.StatusCode, res.Header.Get("Content-Type"), body
}

func TestArtifactPDFErrors(t *testing.T) {
	// This test program is no browser: given a browser's switches it exits with an error.
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	none, report, page := pdfSetup(t, nil)
	broken, brokenReport, _ := pdfSetup(t, &pdf.Browser{Path: self})
	for _, c := range []struct {
		name string
		api  *httptest.Server
		id   string
		want int
	}{
		{"no such artifact", none, "nope", http.StatusNotFound},
		{"not a report", none, page, http.StatusUnprocessableEntity},
		{"no browser", none, report, http.StatusNotImplemented},
		{"browser fails", broken, brokenReport, http.StatusBadGateway},
	} {
		status, contentType, body := postPDF(t, c.api, c.id)
		if status != c.want || contentType != "application/json" || !strings.Contains(string(body), `"detail"`) {
			t.Errorf("%s: got %d %s %s, want %d with a detail", c.name, status, contentType, body, c.want)
		}
	}
}

func TestArtifactPDF(t *testing.T) {
	browser := pdf.Find(os.Getenv("KLEAN_BROWSER"))
	if browser == nil {
		t.Skip("no Chromium-based browser is installed")
	}
	api, report, _ := pdfSetup(t, browser)
	status, contentType, body := postPDF(t, api, report)
	if status != http.StatusOK || contentType != "application/pdf" || !bytes.HasPrefix(body, []byte("%PDF")) {
		t.Errorf("got %d %s %.40q", status, contentType, body)
	}
}
