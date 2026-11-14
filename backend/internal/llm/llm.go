// Package llm talks to any OpenAI-compatible server through the official
// openai-go SDK: GET {base}/models and a streamed reply from POST
// {base}/responses, or from POST {base}/chat/completions on a provider that
// has no Responses API.
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"klean/backend/internal/config"
)

var httpClient = &http.Client{Transport: &http.Transport{
	Proxy:               http.ProxyFromEnvironment,
	DialContext:         (&net.Dialer{Timeout: config.ProviderDialTimeout, KeepAlive: config.ProviderKeepAlive}).DialContext,
	TLSHandshakeTimeout: config.ProviderTLSHandshakeTimeout,
	ForceAttemptHTTP2:   true,
	MaxIdleConns:        config.ProviderMaxIdleConns,
	IdleConnTimeout:     config.ProviderIdleConnTimeout,
}}

// newClient builds an SDK client for one provider. Every setting the SDK would
// otherwise read from OPENAI_* environment variables is overridden, so an
// OpenAI key in the environment is never sent to some other provider.
func newClient(baseURL, apiKey string) openai.Client {
	return openai.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAdminAPIKey(""),
		option.WithOrganization(""),
		option.WithProject(""),
		option.WithAPIKey(apiKey), // "" sends no Authorization header
		option.WithHTTPClient(httpClient),
		option.WithMaxRetries(0), // retrying a reply would generate it twice
	)
}

func endpoint(base, path string) string { return strings.TrimRight(base, "/") + path }

// ListModels returns the provider's models sorted by id.
func ListModels(ctx context.Context, baseURL, apiKey string) ([]Model, error) {
	url := endpoint(baseURL, "/models")
	client := newClient(baseURL, apiKey)
	ctx, cancel := context.WithTimeout(ctx, config.ModelsTimeout)
	defer cancel()

	page, err := client.Models.List(ctx)
	if err != nil {
		return nil, explain(err, url)
	}
	models := make([]Model, 0, len(page.Data))
	for _, m := range page.Data {
		if m.ID != "" {
			models = append(models, Model{ID: m.ID, OwnedBy: m.OwnedBy})
		}
	}
	if len(models) == 0 {
		// Some servers list models under "models" instead of "data".
		models = looseModels(page.RawJSON())
	}
	sort.Slice(models, func(i, j int) bool { return strings.ToLower(models[i].ID) < strings.ToLower(models[j].ID) })
	return models, nil
}

func looseModels(raw string) []Model {
	var body struct {
		Models []json.RawMessage `json:"models"`
	}
	if json.Unmarshal([]byte(raw), &body) != nil {
		return nil
	}
	out := []Model{}
	for _, item := range body.Models {
		var name string
		if json.Unmarshal(item, &name) == nil && name != "" {
			out = append(out, Model{ID: name})
			continue
		}
		var m struct{ ID, Name, Model, OwnedBy string }
		if json.Unmarshal(item, &m) == nil {
			id := firstNonEmpty(m.ID, m.Name, m.Model)
			if id != "" {
				out = append(out, Model{ID: id, OwnedBy: m.OwnedBy})
			}
		}
	}
	return out
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}

// completionsOnly remembers, until the process exits, the base URLs found to
// have no Responses API, so later requests go straight to Chat Completions.
var completionsOnly sync.Map

// StreamReply streams one reply, passing text and the start of tool calls to
// h as they arrive, and returns the tool calls the model made (none when the
// reply is finished). It uses the Responses API, and Chat Completions for a
// provider that turns out not to have it. Cancelling ctx stops generation and
// closes the request to the provider. Errors meant for the user are
// *UpstreamError; a cancelled ctx returns ctx.Err().
func StreamReply(ctx context.Context, r Request, h Handler) ([]ToolCall, error) {
	if _, known := completionsOnly.Load(r.BaseURL); known {
		return streamCompletions(ctx, r, h)
	}
	calls, err := streamResponses(ctx, r, h)
	if !endpointMissing(err) {
		return calls, err
	}
	// Nothing was streamed: the provider refused the request outright.
	calls, err = streamCompletions(ctx, r, h)
	if endpointMissing(err) {
		return nil, upstream("%s has neither %s nor %s. Check that the base URL ends where the API starts (often /v1).",
			r.BaseURL, config.ResponsesPath, config.CompletionsPath)
	}
	completionsOnly.Store(r.BaseURL, true)
	return calls, err
}

var errIdle = errors.New("idle")

// watchIdle gives up on a provider that goes quiet, but not merely because a
// reply is long: the returned context is cancelled once touch hasn't been
// called for config.ReplyIdleTimeout. stop releases it.
func watchIdle(ctx context.Context) (watched context.Context, touch, stop func()) {
	watched, cancel := context.WithCancelCause(ctx)
	timer := time.AfterFunc(config.ReplyIdleTimeout, func() { cancel(errIdle) })
	touch = func() { timer.Reset(config.ReplyIdleTimeout) }
	stop = func() { timer.Stop(); cancel(nil) }
	return watched, touch, stop
}

// endOfStream is the result of a stream that stopped without a final event:
// what arrived when it simply closed, otherwise the reason it broke off.
// ctx is the one from watchIdle.
func endOfStream(ctx context.Context, err error, r Request, path string, gotText bool, calls []ToolCall) ([]ToolCall, error) {
	url := endpoint(r.BaseURL, path)
	if context.Cause(ctx) == errIdle {
		return nil, upstream("The provider stopped responding (timed out).")
	}
	if err != nil {
		return nil, explainRequest(explain(err, url), r)
	}
	if !gotText && len(calls) == 0 {
		return nil, upstream("%s ended without a reply.", url)
	}
	return calls, nil
}
