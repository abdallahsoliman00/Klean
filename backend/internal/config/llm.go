package config

import "time"

// Talking to the model provider.
const (
	ModelsTimeout = 15 * time.Second // listing a provider's models
	// ReplyIdleTimeout abandons a reply when the provider sends nothing for
	// this long. A long reply that keeps streaming is never cut off.
	ReplyIdleTimeout = 300 * time.Second

	ProviderDialTimeout         = 10 * time.Second
	ProviderKeepAlive           = 30 * time.Second
	ProviderTLSHandshakeTimeout = 10 * time.Second
	ProviderMaxIdleConns        = 20
	ProviderIdleConnTimeout     = 90 * time.Second

	// The two APIs a provider may speak, as paths under its base URL.
	// /responses is tried first; a provider without it gets /chat/completions.
	ResponsesPath   = "/responses"
	CompletionsPath = "/chat/completions"

	ErrorBodyBytes   = 4096 // read from an error response
	ErrorDetailChars = 300  // of an error body shown to the user

	// MaxToolRounds is the most requests one reply may take: each time the
	// model calls tools, they run and the model is asked again, so a model
	// that keeps calling tools stops here.
	MaxToolRounds = 8

	// MaxWebSearchToolRounds is the round limit for a reply that reaches the
	// web: searching and reading pages takes more requests than other tools.
	MaxWebSearchToolRounds = 20
)
