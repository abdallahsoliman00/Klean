package config

import "time"

// web_search.
const (
	MaxSearchResults   = 6
	SearchTimeout      = 20 * time.Second
	MaxSearchBody      = 2 << 20 // bytes read from a search reply
	SearchSnippetChars = 400

	DuckDuckGoURL = "https://html.duckduckgo.com/html/"
	BraveURL      = "https://api.search.brave.com/res/v1/web/search"
	TavilyURL     = "https://api.tavily.com/search"
)

// fetch_page.
const (
	FetchTimeout               = 20 * time.Second // the whole request
	FetchDialTimeout           = 10 * time.Second
	FetchTLSHandshakeTimeout   = 10 * time.Second
	FetchResponseHeaderTimeout = 15 * time.Second
	FetchMaxIdleConns          = 4
	FetchIdleConnTimeout       = 30 * time.Second
	MaxRedirects               = 5

	MaxPageBody    = 3 << 20 // bytes read from a page
	MaxPageChars   = 12000   // characters of text handed to the model
	PageTitleChars = 200
	// MinReadableChars: when leaving out navigation, footers and the like
	// leaves less text than this, the whole page is kept instead.
	MinReadableChars = 200
)

// Request headers for search and fetch.
const (
	// A browser-like User-Agent: DuckDuckGo's HTML endpoint and many sites
	// turn away clients that announce themselves as a library.
	UserAgent      = "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"
	AcceptLanguage = "en-US,en;q=0.8"
)

// Tools carried out by the server.
const (
	MaxAskOptions      = 4   // suggested answers ask_user may offer
	ArtifactTitleChars = 200 // longer artifact titles are cut

	// How much of a bad URL or content type is quoted back in an error.
	ErrorURLChars         = 200
	ErrorContentTypeChars = 80
)
