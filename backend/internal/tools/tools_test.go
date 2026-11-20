package tools

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"slices"
	"strings"
	"testing"

	"klean/backend/internal/config"
)

func TestBlockedAddresses(t *testing.T) {
	for addr, want := range map[string]bool{
		"127.0.0.1": true, "10.1.2.3": true, "192.168.1.1": true, "172.16.0.1": true, "169.254.169.254": true,
		"0.0.0.0": true, "100.64.0.1": true, "::1": true, "fe80::1": true, "fd00::1": true, "::ffff:127.0.0.1": true,
		"::ffff:10.0.0.1": true, "64:ff9b::7f00:1": true, "224.0.0.1": true,
		"93.184.216.34": false, "2606:2800:220:1:248:1893:25c8:1946": false, "172.32.0.1": false,
	} {
		if got := blocked(netip.MustParseAddr(addr)); got != want {
			t.Errorf("blocked(%s) = %v, want %v", addr, got, want)
		}
	}
}

func TestFetchRefusals(t *testing.T) {
	w := NewWeb()
	for _, u := range []string{
		"file:///etc/passwd", "ftp://example.com/x", "javascript:alert(1)", "example.com/page", "",
		"http://localhost:8787/api/providers", "http://127.0.0.1/", "http://[::1]:8080/", "http://192.168.0.1/admin",
		"http://169.254.169.254/latest/meta-data/", "http://user:pw@example.com/", "http://app.localhost/",
	} {
		_, err := w.Fetch(context.Background(), u)
		var te *Error
		if !errors.As(err, &te) {
			t.Errorf("Fetch(%q) = %v, want a refusal", u, err)
		}
	}
}

// The check at connection time is what stops redirects and DNS names that
// point at local addresses; here it is exercised without the URL pre-check.
func TestDialGuard(t *testing.T) {
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<p>internal</p>")
	}))
	defer local.Close()
	if _, err := newFetchClient(false).Get(local.URL); !errors.Is(err, errBlockedAddress) {
		t.Fatalf("guarded client reached a loopback server: %v", err)
	}
	// With the hook, a redirect is followed and the final URL reported.
	hop := httptest.NewServer(http.RedirectHandler(local.URL+"/target", http.StatusFound))
	defer hop.Close()
	w := NewWeb()
	w.AllowPrivate = true
	page, err := w.Fetch(context.Background(), hop.URL)
	if err != nil || page.URL != local.URL+"/target" || page.Text != "internal" {
		t.Fatalf("redirected fetch: %+v, %v", page, err)
	}
}

func TestFetchLimitsAndTypes(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/long", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><body><p>"+strings.Repeat("word ", 10000)+"</p></body></html>")
	})
	mux.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 'P', 'N', 'G'})
	})
	mux.HandleFunc("/plain", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "just <text>")
	})
	mux.HandleFunc("/latin1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=iso-8859-1")
		_, _ = w.Write([]byte("<p>caf\xe9</p>"))
	})
	mux.HandleFunc("/empty", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><body><script>render()</script></body></html>")
	})
	mux.HandleFunc("/furniture", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<html><body><footer>All of it is in the footer</footer></body></html>")
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	w := NewWeb()
	w.AllowPrivate = true
	ctx := context.Background()

	page, err := w.Fetch(ctx, srv.URL+"/long")
	if err != nil || !page.Cut || page.Chars != config.MaxPageChars || !strings.Contains(FormatPage(page), "page continues") {
		t.Fatalf("long page: chars=%d cut=%v err=%v", page.Chars, page.Cut, err)
	}
	if _, err = w.Fetch(ctx, srv.URL+"/image"); err == nil || !strings.Contains(err.Error(), "image/png") {
		t.Fatalf("image: %v", err)
	}
	if page, err = w.Fetch(ctx, srv.URL+"/plain"); err != nil || page.Text != "just <text>" {
		t.Fatalf("plain text: %+v %v", page, err)
	}
	if page, err = w.Fetch(ctx, srv.URL+"/latin1"); err != nil || page.Text != "café" {
		t.Fatalf("latin-1: %+v %v", page, err)
	}
	if _, err = w.Fetch(ctx, srv.URL+"/empty"); err == nil || !strings.Contains(err.Error(), "no readable text") {
		t.Fatalf("empty page: %v", err)
	}
	if page, err = w.Fetch(ctx, srv.URL+"/furniture"); err != nil || page.Text != "All of it is in the footer" {
		t.Fatalf("furniture-only page: %+v %v", page, err)
	}
	if _, err = w.Fetch(ctx, srv.URL+"/missing"); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("404: %v", err)
	}
}

func TestSearchProviders(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ddg", func(w http.ResponseWriter, r *http.Request) {
		switch r.FormValue("q") {
		case "nothing":
			fmt.Fprint(w, `<html><div class="no-results">No results.</div></html>`)
		case "many":
			for i := 0; i < 10; i++ {
				fmt.Fprintf(w, `<a class="result__a" href="https://example.com/%d">Result %d</a>`, i, i)
			}
		default:
			w.WriteHeader(403)
		}
	})
	mux.HandleFunc("/tavily", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tv-key" {
			w.WriteHeader(401)
			return
		}
		fmt.Fprint(w, `{"results":[{"title":"T","url":"https://t.example/","content":"body text"}]}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	w := NewWeb()
	w.DuckDuckGoURL, w.TavilyURL = srv.URL+"/ddg", srv.URL+"/tavily"
	ctx := context.Background()

	if res, err := w.Search(ctx, DuckDuckGo, "", "many"); err != nil || len(res) != config.MaxSearchResults || res[5].URL != "https://example.com/5" {
		t.Fatalf("many: %d results, %v", len(res), err)
	}
	if _, err := w.Search(ctx, DuckDuckGo, "", "nothing"); err == nil || !strings.Contains(err.Error(), "No results") {
		t.Fatalf("no results: %v", err)
	}
	if _, err := w.Search(ctx, DuckDuckGo, "", "anything"); err == nil || !strings.Contains(err.Error(), "turned the search away") {
		t.Fatalf("blocked: %v", err)
	}
	if _, err := w.Search(ctx, DuckDuckGo, "", "  "); err == nil {
		t.Fatal("empty query accepted")
	}
	if res, err := w.Search(ctx, Tavily, "tv-key", "q"); err != nil || len(res) != 1 || res[0].Snippet != "body text" {
		t.Fatalf("tavily: %+v %v", res, err)
	}
	if _, err := w.Search(ctx, Tavily, "wrong", "q"); err == nil || !strings.Contains(err.Error(), "rejected the API key") {
		t.Fatalf("tavily bad key: %v", err)
	}
}

func TestDefsAndGuidance(t *testing.T) {
	if n := len(Defs(AllGroups)); n != 7 {
		t.Fatalf("%d tools", n)
	}
	if Defs(nil) != nil || Guidance(nil) != "" {
		t.Fatal("no groups should mean no tools and no guidance")
	}
	for _, d := range append(Defs(AllGroups), SkillDef) {
		if d.Description == "" || d.Parameters["type"] != "object" {
			t.Errorf("%s: incomplete definition", d.Name)
		}
	}
}

// A report is only made through create_report: the generic tool doesn't offer the kind.
func TestReportHasItsOwnTool(t *testing.T) {
	var names []string
	for _, d := range Defs([]string{GroupArtifacts}) {
		names = append(names, d.Name)
		if d.Name != CreateArtifact {
			continue
		}
		kinds := d.Parameters["properties"].(obj)["kind"].(obj)["enum"].([]string)
		if slices.Contains(kinds, KindReport) {
			t.Fatalf("create_artifact offers the report kind: %v", kinds)
		}
	}
	if got := strings.Join(names, ","); got != "create_artifact,create_report,update_artifact,edit_artifact" {
		t.Fatalf("artifact tools: %s", got)
	}
	if g := Guidance([]string{GroupArtifacts}); !strings.Contains(g, CreateReport) || !strings.Contains(g, "A4") {
		t.Fatalf("guidance doesn't cover reports: %q", g)
	}
}

func TestSkillGuidance(t *testing.T) {
	if SkillGuidance(nil) != "" {
		t.Fatal("guidance without skills")
	}
	g := SkillGuidance([]Skill{{"weekly-report", "Use for the weekly status report."}, {"tone", "How to write to customers."}})
	for _, want := range []string{UseSkill, "\n- weekly-report: Use for the weekly status report.", "\n- tone: How to write to customers."} {
		if !strings.Contains(g, want) {
			t.Errorf("guidance lacks %q: %q", want, g)
		}
	}
}
