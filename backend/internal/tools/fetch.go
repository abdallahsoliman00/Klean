package tools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html/charset"
)

const (
	maxPageBody  = 3 << 20 // bytes read from a page
	maxPageChars = 12000   // characters of text handed to the model
	maxRedirects = 5
)

// Page is what fetch_page returns.
type Page struct {
	URL   string // after redirects
	Title string
	Text  string // readable text, at most maxPageChars
	Chars int    // length of Text in characters
	Cut   bool   // the page had more text than was returned
}

var errBlockedAddress = errors.New("blocked address")

// blockedRanges are not covered by the netip.Addr predicates used in blocked.
var blockedRanges = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"), // carrier-grade NAT, used by VPN meshes
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"), // NAT64: an IPv4 address in disguise
}

// blocked reports whether addr is on this machine or a private network.
func blocked(addr netip.Addr) bool {
	addr = addr.Unmap()
	if !addr.IsValid() || addr.IsLoopback() || addr.IsPrivate() || addr.IsUnspecified() ||
		addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsMulticast() {
		return true
	}
	for _, p := range blockedRanges {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// guardDial runs just before each connection is made, with the address the
// name resolved to. Checking here rather than before the request covers
// redirects and names that resolve to something different the second time.
func guardDial(network, address string, _ syscall.RawConn) error {
	ap, err := netip.ParseAddrPort(address)
	if err != nil || blocked(ap.Addr()) {
		return errBlockedAddress
	}
	return nil
}

func newFetchClient(allowPrivate bool) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	if !allowPrivate {
		dialer.Control = guardDial
	}
	return &http.Client{
		Timeout: fetchTimeout,
		Transport: &http.Transport{
			// No proxy: through one, the address check above would see the
			// proxy's address instead of the page's.
			Proxy:                 nil,
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 15 * time.Second,
			MaxIdleConns:          4,
			IdleConnTimeout:       30 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return errBlockedAddress
			}
			return nil
		},
	}
}

const privateMsg = "That address is on this computer or a private network, which fetch_page is not allowed to reach."

// checkURL rejects what can be rejected before connecting, with a clearer
// message than the dial-time check can give.
func (w *Web) checkURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return nil, failf("%q is not a full URL. It needs to start with http:// or https://.", clip(raw, 200))
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, failf("Only http and https URLs can be fetched, not %s.", u.Scheme)
	}
	if u.User != nil {
		return nil, failf("URLs with a username or password in them are not fetched.")
	}
	if w.AllowPrivate {
		return u, nil
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return nil, failf("%s", privateMsg)
	}
	if addr, err := netip.ParseAddr(host); err == nil && blocked(addr) {
		return nil, failf("%s", privateMsg)
	}
	return u, nil
}

// Fetch downloads a page and extracts its readable text.
func (w *Web) Fetch(ctx context.Context, raw string) (*Page, error) {
	u, err := w.checkURL(raw)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, failf("%q is not a URL that can be fetched.", clip(raw, 200))
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain;q=0.9,*/*;q=0.5")
	req.Header.Set("Accept-Language", "en-US,en;q=0.8")

	w.clients()
	res, err := w.fetch.Do(req)
	if err != nil {
		return nil, fetchError(ctx, err, u)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, failf("%s answered %d %s.", u.Host, res.StatusCode, http.StatusText(res.StatusCode))
	}
	ctype := strings.ToLower(res.Header.Get("Content-Type"))
	isHTML := ctype == "" || strings.Contains(ctype, "html")
	if !isHTML && !isText(ctype) {
		return nil, failf("That URL is %s, not a page of text, so it can't be read.", clip(ctype, 80))
	}
	// Convert to UTF-8 using the header, then a <meta> tag, then a guess.
	body, err := charset.NewReader(io.LimitReader(res.Body, maxPageBody), res.Header.Get("Content-Type"))
	if err != nil {
		return nil, failf("Reading %s failed.", u.Host)
	}
	data, err := io.ReadAll(body)
	if err != nil && len(data) == 0 {
		return nil, fetchError(ctx, err, u)
	}

	page := &Page{URL: res.Request.URL.String()}
	if isHTML {
		page.Title, page.Text = readable(data)
	} else {
		page.Text = strings.TrimSpace(strings.ToValidUTF8(string(data), ""))
	}
	if page.Text == "" {
		return nil, failf("The page at %s has no readable text (it may need JavaScript to show anything).", u.Host)
	}
	if utf8.RuneCountInString(page.Text) > maxPageChars {
		page.Text, page.Cut = string([]rune(page.Text)[:maxPageChars]), true
	}
	page.Chars = utf8.RuneCountInString(page.Text)
	return page, nil
}

func isText(ctype string) bool {
	for _, t := range []string{"text/", "json", "xml", "javascript", "yaml", "csv"} {
		if strings.Contains(ctype, t) {
			return true
		}
	}
	return false
}

func fetchError(ctx context.Context, err error, u *url.URL) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if errors.Is(err, errBlockedAddress) {
		return failf("%s", privateMsg)
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return failf("%s took too long to answer.", u.Host)
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return failf("The address %s could not be found.", u.Hostname())
	}
	if strings.Contains(err.Error(), "too many redirects") {
		return failf("%s redirected too many times.", u.Host)
	}
	return failf("Couldn't connect to %s.", u.Host)
}

// FormatPage is the text the model reads.
func FormatPage(p *Page) string {
	var b strings.Builder
	if p.Title != "" {
		fmt.Fprintf(&b, "Title: %s\n", p.Title)
	}
	fmt.Fprintf(&b, "URL: %s\n\n%s", p.URL, p.Text)
	if p.Cut {
		fmt.Fprintf(&b, "\n\n[The page continues; only the first %d characters are shown.]", maxPageChars)
	}
	return b.String()
}
