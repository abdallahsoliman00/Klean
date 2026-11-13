package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/net/html"

	"klean/backend/internal/config"
)

// Search providers, as saved in settings.
const (
	DuckDuckGo = "duckduckgo"
	Brave      = "brave"
	Tavily     = "tavily"
)

func ValidSearchProvider(p string) bool { return p == DuckDuckGo || p == Brave || p == Tavily }

// Error is a failure worth telling the model and the user about in its own words.
type Error struct{ Msg string }

func (e *Error) Error() string { return e.Msg }

func failf(format string, a ...any) error { return &Error{fmt.Sprintf(format, a...)} }

type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// Web runs web_search and fetch_page. The endpoints are fields so tests can
// point them at local stubs.
type Web struct {
	DuckDuckGoURL string
	BraveURL      string
	TavilyURL     string
	// AllowPrivate lets fetch_page reach loopback and private addresses.
	// Only tests set it: the URL comes from the model, which may be repeating
	// instructions from a web page, and must not be able to reach services on
	// this machine or the local network.
	AllowPrivate bool

	once   sync.Once
	search *http.Client
	fetch  *http.Client
}

func NewWeb() *Web {
	return &Web{
		DuckDuckGoURL: config.DuckDuckGoURL,
		BraveURL:      config.BraveURL,
		TavilyURL:     config.TavilyURL,
	}
}

// clients builds the two HTTP clients on first use, once the fields are set.
func (w *Web) clients() {
	w.once.Do(func() {
		w.search = &http.Client{Timeout: config.SearchTimeout}
		w.fetch = newFetchClient(w.AllowPrivate)
	})
}

// Search returns the top results from the chosen provider. key is ignored for DuckDuckGo.
func (w *Web) Search(ctx context.Context, provider, key, query string) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, failf("The search query is empty.")
	}
	var (
		results []Result
		err     error
	)
	switch provider {
	case Brave, Tavily:
		if key == "" {
			return nil, failf("Web search is set to %s but no API key is saved. Add one in settings, or switch to DuckDuckGo.", provider)
		}
		if provider == Brave {
			results, err = w.brave(ctx, key, query)
		} else {
			results, err = w.tavily(ctx, key, query)
		}
	default:
		results, err = w.duckDuckGo(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	if len(results) > config.MaxSearchResults {
		results = results[:config.MaxSearchResults]
	}
	return results, nil
}

// do performs a search request and returns the body, turning transport
// failures and error statuses into messages that name the search service.
func (w *Web) do(req *http.Request, service string) ([]byte, int, error) {
	w.clients()
	res, err := w.search.Do(req)
	if err != nil {
		if ctxErr := req.Context().Err(); ctxErr != nil {
			return nil, 0, ctxErr
		}
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return nil, 0, failf("%s did not answer in time.", service)
		}
		return nil, 0, failf("Couldn't reach %s. Check the internet connection.", service)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, config.MaxSearchBody))
	if err != nil {
		return nil, 0, failf("Reading the reply from %s failed.", service)
	}
	return body, res.StatusCode, nil
}

func keyError(service string, status int) error {
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return failf("%s rejected the API key (%d). Check the key in settings.", service, status)
	}
	if status == http.StatusTooManyRequests || status == 432 {
		return failf("%s says the search quota is used up (%d).", service, status)
	}
	return failf("%s returned %d.", service, status)
}

// ---------------- DuckDuckGo ----------------

const ddgBlocked = "DuckDuckGo turned the search away (it limits automated requests). " +
	"Try again in a little while, or set a Brave or Tavily key in settings for reliable search."

func (w *Web) duckDuckGo(ctx context.Context, query string) ([]Result, error) {
	form := url.Values{"q": {query}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.DuckDuckGoURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", config.UserAgent)
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Accept-Language", config.AcceptLanguage)
	body, status, err := w.do(req, "DuckDuckGo")
	if err != nil {
		return nil, err
	}
	results := parseDuckDuckGo(body)
	if len(results) > 0 {
		return results, nil
	}
	// A challenge page comes back as 202 or 403, or as a 200 with an "anomaly" form.
	low := bytes.ToLower(body)
	if status != http.StatusOK || bytes.Contains(low, []byte("anomaly")) || bytes.Contains(low, []byte("challenge")) {
		return nil, failf("%s", ddgBlocked)
	}
	if bytes.Contains(low, []byte("no results")) || bytes.Contains(low, []byte("no-results")) {
		return nil, failf("No results for %q.", query)
	}
	return nil, failf("DuckDuckGo returned a page with no results for %q. It may be limiting automated requests; "+
		"try again, or set a Brave or Tavily key in settings.", query)
}

// parseDuckDuckGo reads the results out of the HTML page: each is a link with
// class result__a, followed by an element with class result__snippet. It goes
// by those two classes only, so changes to the surrounding markup don't matter.
func parseDuckDuckGo(page []byte) []Result {
	doc, err := html.Parse(bytes.NewReader(page))
	if err != nil {
		return nil
	}
	var out []Result
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch {
			case n.Data == "a" && hasClass(n, "result__a"):
				if u := ddgTarget(attr(n, "href")); u != "" {
					if title := collapse(nodeText(n)); title != "" {
						out = append(out, Result{Title: title, URL: u})
					}
				}
				return
			case hasClass(n, "result__snippet"):
				if len(out) > 0 && out[len(out)-1].Snippet == "" {
					out[len(out)-1].Snippet = collapse(nodeText(n))
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out
}

// ddgTarget unwraps DuckDuckGo's redirect links (//duckduckgo.com/l/?uddg=…)
// and drops adverts, which go through /y.js.
func ddgTarget(href string) string {
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "//") {
		href = "https:" + href
	}
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if strings.HasSuffix(u.Hostname(), "duckduckgo.com") || u.Host == "" {
		if strings.HasPrefix(u.Path, "/y.js") {
			return ""
		}
		target := u.Query().Get("uddg")
		if target == "" {
			return ""
		}
		if u, err = url.Parse(target); err != nil {
			return ""
		}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return u.String()
}

// ---------------- Brave ----------------

func (w *Web) brave(ctx context.Context, key, query string) ([]Result, error) {
	q := url.Values{"q": {query}, "count": {fmt.Sprint(config.MaxSearchResults)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.BraveURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", key)
	body, status, err := w.do(req, "Brave Search")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, keyError("Brave Search", status)
	}
	var parsed struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, failf("Brave Search sent a reply that couldn't be read.")
	}
	var out []Result
	for _, r := range parsed.Web.Results {
		// Titles and descriptions carry <strong> highlights and HTML entities.
		out = append(out, Result{Title: plainText(r.Title), URL: r.URL, Snippet: plainText(r.Description)})
	}
	if len(out) == 0 {
		return nil, failf("No results for %q.", query)
	}
	return out, nil
}

// ---------------- Tavily ----------------

func (w *Web) tavily(ctx context.Context, key, query string) ([]Result, error) {
	payload, _ := json.Marshal(map[string]any{"query": query, "max_results": config.MaxSearchResults})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.TavilyURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	body, status, err := w.do(req, "Tavily")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, keyError("Tavily", status)
	}
	var parsed struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, failf("Tavily sent a reply that couldn't be read.")
	}
	var out []Result
	for _, r := range parsed.Results {
		out = append(out, Result{Title: collapse(r.Title), URL: r.URL, Snippet: clip(collapse(r.Content), config.SearchSnippetChars)})
	}
	if len(out) == 0 {
		return nil, failf("No results for %q.", query)
	}
	return out, nil
}

// FormatResults is the text the model reads.
func FormatResults(query string, results []Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Search results for %q:\n", query)
	for i, r := range results {
		fmt.Fprintf(&b, "\n%d. %s\n   %s\n", i+1, r.Title, r.URL)
		if r.Snippet != "" {
			fmt.Fprintf(&b, "   %s\n", r.Snippet)
		}
	}
	return b.String()
}
