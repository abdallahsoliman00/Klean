// Package llm talks to any OpenAI-compatible server through the official
// openai-go SDK: GET {base}/models and streaming POST {base}/responses.
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/responses"

	"switchboard/backend/internal/prompt"
)

const (
	modelsTimeout = 15 * time.Second
	// The reply is abandoned if the provider sends nothing for this long.
	idleTimeout = 300 * time.Second
)

// UpstreamError carries a message meant for the person using the UI.
type UpstreamError struct{ Msg string }

func (e *UpstreamError) Error() string { return e.Msg }

func upstream(format string, a ...any) error { return &UpstreamError{fmt.Sprintf(format, a...)} }

type Model struct {
	ID      string `json:"id"`
	OwnedBy string `json:"ownedBy"`
}

var httpClient = &http.Client{Transport: &http.Transport{
	Proxy:               http.ProxyFromEnvironment,
	DialContext:         (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
	TLSHandshakeTimeout: 10 * time.Second,
	ForceAttemptHTTP2:   true,
	MaxIdleConns:        20,
	IdleConnTimeout:     90 * time.Second,
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
	ctx, cancel := context.WithTimeout(ctx, modelsTimeout)
	defer cancel()

	page, err := client.Models.List(ctx)
	if err != nil {
		return nil, explain(err, url, false)
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

// Request is one reply to generate.
type Request struct {
	BaseURL      string
	APIKey       string
	Model        string
	Instructions string // the chat's system prompt
	Turns        []prompt.Turn
	Temperature  float64
}

// StreamReply streams a reply through the Responses API, calling onDelta with
// each piece of text as it arrives. Cancelling ctx stops generation and closes
// the request to the provider. Errors meant for the user are *UpstreamError;
// a cancelled ctx returns ctx.Err().
func StreamReply(ctx context.Context, r Request, onDelta func(string)) error {
	url := endpoint(r.BaseURL, "/responses")
	client := newClient(r.BaseURL, r.APIKey)

	input := make(responses.ResponseInputParam, 0, len(r.Turns))
	for _, t := range r.Turns {
		role := responses.EasyInputMessageRoleUser
		if t.Role == "assistant" {
			role = responses.EasyInputMessageRoleAssistant
		}
		input = append(input, responses.ResponseInputItemParamOfMessage(t.Content, role))
	}
	params := responses.ResponseNewParams{
		Model:       r.Model,
		Input:       responses.ResponseNewParamsInputUnion{OfInputItemList: input},
		Temperature: openai.Float(r.Temperature),
		// The history lives in our database; don't keep a copy on the provider.
		Store: openai.Bool(false),
	}
	if r.Instructions != "" {
		params.Instructions = openai.String(r.Instructions)
	}

	// Give up if the provider goes quiet, but not merely because a reply is long.
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	errIdle := errors.New("idle")
	idle := time.AfterFunc(idleTimeout, func() { cancel(errIdle) })
	defer idle.Stop()

	stream := client.Responses.NewStreaming(ctx, params)
	defer stream.Close()

	got := false
	for stream.Next() {
		idle.Reset(idleTimeout)
		ev := stream.Current()
		switch ev.Type {
		case "response.output_text.delta", "response.refusal.delta":
			if ev.Delta != "" {
				got = true
				onDelta(ev.Delta)
			}
		case "response.completed", "response.incomplete":
			return nil
		case "response.failed":
			if msg := ev.Response.Error.Message; msg != "" {
				return upstream("The provider failed to answer: %s", msg)
			}
			return upstream("The provider failed to answer.")
		case "error":
			if ev.Message != "" {
				return upstream("The provider reported an error: %s", ev.Message)
			}
			return upstream("The provider reported an error.")
		}
	}
	err := stream.Err()
	if context.Cause(ctx) == errIdle {
		return upstream("The provider stopped responding (timed out).")
	}
	if err == nil {
		if !got {
			return upstream("%s ended without a reply. Check that this provider supports the Responses API.", url)
		}
		return nil // stream closed without a final event; keep what arrived
	}
	return explain(err, url, true)
}

// explain turns an SDK or network error into a message for the user.
func explain(err error, url string, isResponses bool) error {
	if errors.Is(err, context.Canceled) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return upstream("The provider stopped responding (timed out).")
	}
	var se *ssestream.StreamError
	if errors.As(err, &se) {
		return upstream("The provider reported an error: %s", errorDetail([]byte(se.Event.Data)))
	}
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusUnauthorized:
			return upstream("The provider rejected the API key (401). Check the key in Providers & settings.")
		case http.StatusNotFound:
			if isResponses {
				return upstream("%s returned 404. Check that the base URL ends where the API starts (often /v1) "+
					"and that this provider supports the Responses API.", url)
			}
			return upstream("%s returned 404. Check that the base URL ends where the API starts (often /v1).", url)
		}
		detail := apiErr.Message
		if detail == "" && apiErr.Response != nil && apiErr.Response.Body != nil {
			body, _ := io.ReadAll(io.LimitReader(apiErr.Response.Body, 4096))
			detail = errorDetail(body)
		}
		if detail != "" {
			return upstream("The provider returned %d: %s", apiErr.StatusCode, detail)
		}
		return upstream("The provider returned %d.", apiErr.StatusCode)
	}
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return upstream("%s didn't return JSON. Is this an OpenAI-compatible endpoint?", url)
	}
	return upstream("Couldn't reach %s. Check that the server is running and the URL is right.", url)
}

// errorDetail pulls a readable message out of an error body.
func errorDetail(body []byte) string {
	var v map[string]any
	if json.Unmarshal(body, &v) == nil {
		for _, key := range []string{"error", "detail", "message"} {
			switch e := v[key].(type) {
			case string:
				return e
			case map[string]any:
				if msg, ok := e["message"].(string); ok {
					return msg
				}
			}
		}
	}
	s := strings.TrimSpace(string(body))
	if r := []rune(s); len(r) > 300 {
		s = string(r[:300])
	}
	return s
}
