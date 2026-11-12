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

	"klean/backend/internal/prompt"
)

const (
	modelsTimeout = 15 * time.Second
	// The reply is abandoned if the provider sends nothing for this long.
	idleTimeout = 300 * time.Second
)

// UpstreamError carries a message meant for the person using the UI.
type UpstreamError struct {
	Msg string
	// rejected is set when the provider refused the request itself (a 4xx or
	// 501), as opposed to failing while answering it.
	rejected bool
}

func (e *UpstreamError) Error() string { return e.Msg }

func upstream(format string, a ...any) error { return &UpstreamError{Msg: fmt.Sprintf(format, a...)} }

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

// Tool is a function the model may call.
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON schema of the arguments object
}

// ToolCall is a function call the model finished writing.
type ToolCall struct {
	ID        string // sent back with the output so the model can match them up
	Name      string
	Arguments string // JSON object, as the model wrote it
}

// Request is one reply to generate.
type Request struct {
	BaseURL      string
	APIKey       string
	Model        string
	Instructions string // the chat's system prompt
	Turns        []prompt.Turn
	Temperature  float64
	Tools        []Tool // none: the request carries no tools at all
}

// Handler receives the reply as it streams. Either function may be nil.
type Handler struct {
	OnDelta     func(text string)
	OnCallStart func(id, name string) // a tool call began; its arguments are still streaming
}

// buildInput turns the history into Responses input items. Nothing is stored
// on the provider, so earlier tool calls are replayed in full: each as a
// function_call item followed by its function_call_output.
func buildInput(turns []prompt.Turn) responses.ResponseInputParam {
	input := make(responses.ResponseInputParam, 0, len(turns))
	for _, t := range turns {
		if c := t.Call; c != nil {
			out := responses.ResponseInputItemParamOfFunctionCallOutput(c.Output)
			out.OfFunctionCallOutput.CallID = openai.String(c.ID)
			input = append(input, responses.ResponseInputItemParamOfFunctionCall(c.Arguments, c.ID, c.Name), out)
			continue
		}
		role := responses.EasyInputMessageRoleUser
		if t.Role == "assistant" {
			role = responses.EasyInputMessageRoleAssistant
		}
		input = append(input, responses.ResponseInputItemParamOfMessage(t.Content, role))
	}
	return input
}

func buildTools(tools []Tool) []responses.ToolUnionParam {
	out := make([]responses.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		// Not strict: strict mode forbids optional arguments, and several
		// OpenAI-compatible servers don't implement it.
		p := responses.ToolParamOfFunction(t.Name, t.Parameters, false)
		p.OfFunction.Description = openai.String(t.Description)
		out = append(out, p)
	}
	return out
}

// rawItem is an output item as it appears in stream events and in the final
// response. It is decoded by hand to accept what looser servers send.
type rawItem struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	CallID    string          `json:"call_id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// arguments returns the JSON text; a few servers send an object instead of a string.
func (it rawItem) arguments() string {
	var s string
	if json.Unmarshal(it.Arguments, &s) == nil {
		return s
	}
	if string(it.Arguments) == "null" {
		return ""
	}
	return string(it.Arguments)
}

type rawEvent struct {
	ItemID      string  `json:"item_id"`
	OutputIndex *int64  `json:"output_index"`
	Name        string  `json:"name"`
	Arguments   string  `json:"arguments"`
	Item        rawItem `json:"item"`
	Response    struct {
		Output []rawItem `json:"output"`
	} `json:"response"`
}

// callTracker pieces tool calls together from the stream. Servers differ in
// which events they send (output_item.added, function_call_arguments.done,
// output_item.done, or only the final response), so a call is taken from
// whichever source has it and matched up so it is never returned twice.
type callTracker struct {
	calls   []*trackedCall
	onStart func(id, name string)
}

type trackedCall struct {
	itemID, callID, name, args string
	index                      int64  // output_index, or -1
	announced                  string // the id given to onStart
	complete                   bool
}

// find returns the call an event refers to, adding it when it is new.
func (t *callTracker) find(itemID, callID string, index *int64) *trackedCall {
	for _, c := range t.calls {
		switch {
		case itemID != "" && c.itemID != "":
			if itemID == c.itemID {
				return c
			}
		case callID != "" && c.callID != "":
			if callID == c.callID {
				return c
			}
		case index != nil && c.index == *index:
			return c
		}
	}
	c := &trackedCall{itemID: itemID, callID: callID, index: -1}
	if index != nil {
		c.index = *index
	}
	t.calls = append(t.calls, c)
	return c
}

func (t *callTracker) item(it rawItem, index *int64, done bool) {
	if it.Type != "function_call" {
		return
	}
	c := t.find(it.ID, it.CallID, index)
	if c.callID == "" {
		c.callID = it.CallID
	}
	if c.name == "" {
		c.name = it.Name
	}
	if a := it.arguments(); a != "" && (done || c.args == "") {
		c.args = a
	}
	c.complete = c.complete || done
	t.announce(c)
}

// announce reports the call once, as soon as its name is known.
func (t *callTracker) announce(c *trackedCall) {
	if c.announced != "" || c.name == "" {
		return
	}
	c.announced = firstNonEmpty(c.callID, c.itemID, fmt.Sprintf("call_%d", len(t.calls)))
	if t.onStart != nil {
		t.onStart(c.announced, c.name)
	}
}

func (t *callTracker) argumentsDone(ev rawEvent) {
	c := t.find(ev.ItemID, "", ev.OutputIndex)
	if c.name == "" {
		c.name = ev.Name
	}
	c.args, c.complete = ev.Arguments, true
	t.announce(c)
}

// finished returns the completed calls in the order the model made them.
// The id is the one announced at the start, so the caller sees one id per call.
func (t *callTracker) finished() []ToolCall {
	var out []ToolCall
	for _, c := range t.calls {
		if !c.complete || c.name == "" {
			continue
		}
		t.announce(c)
		out = append(out, ToolCall{ID: c.announced, Name: c.name, Arguments: c.args})
	}
	return out
}

// StreamReply streams one response through the Responses API, passing text
// and the start of tool calls to h as they arrive, and returns the tool calls
// the model made (none when the reply is finished). Cancelling ctx stops
// generation and closes the request to the provider. Errors meant for the
// user are *UpstreamError; a cancelled ctx returns ctx.Err().
func StreamReply(ctx context.Context, r Request, h Handler) ([]ToolCall, error) {
	url := endpoint(r.BaseURL, "/responses")
	client := newClient(r.BaseURL, r.APIKey)

	params := responses.ResponseNewParams{
		Model:       r.Model,
		Input:       responses.ResponseNewParamsInputUnion{OfInputItemList: buildInput(r.Turns)},
		Temperature: openai.Float(r.Temperature),
		// The history lives in our database; don't keep a copy on the provider.
		Store: openai.Bool(false),
	}
	if r.Instructions != "" {
		params.Instructions = openai.String(r.Instructions)
	}
	if len(r.Tools) > 0 {
		params.Tools = buildTools(r.Tools)
	}

	// Give up if the provider goes quiet, but not merely because a reply is long.
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	errIdle := errors.New("idle")
	idle := time.AfterFunc(idleTimeout, func() { cancel(errIdle) })
	defer idle.Stop()

	stream := client.Responses.NewStreaming(ctx, params)
	defer stream.Close()

	tracker := &callTracker{onStart: h.OnCallStart}
	got := false
	for stream.Next() {
		idle.Reset(idleTimeout)
		ev := stream.Current()
		switch ev.Type {
		case "response.output_text.delta", "response.refusal.delta":
			if ev.Delta != "" {
				got = true
				if h.OnDelta != nil {
					h.OnDelta(ev.Delta)
				}
			}
		case "response.output_item.added", "response.output_item.done":
			var raw rawEvent
			if json.Unmarshal([]byte(ev.RawJSON()), &raw) == nil {
				tracker.item(raw.Item, raw.OutputIndex, ev.Type == "response.output_item.done")
			}
		case "response.function_call_arguments.done":
			var raw rawEvent
			if json.Unmarshal([]byte(ev.RawJSON()), &raw) == nil {
				tracker.argumentsDone(raw)
			}
		case "response.completed", "response.incomplete":
			var raw rawEvent
			if json.Unmarshal([]byte(ev.RawJSON()), &raw) == nil {
				for i, it := range raw.Response.Output {
					index := int64(i)
					tracker.item(it, &index, true)
				}
			}
			return tracker.finished(), nil
		case "response.failed":
			if msg := ev.Response.Error.Message; msg != "" {
				return nil, upstream("The provider failed to answer: %s", msg)
			}
			return nil, upstream("The provider failed to answer.")
		case "error":
			if ev.Message != "" {
				return nil, upstream("The provider reported an error: %s", ev.Message)
			}
			return nil, upstream("The provider reported an error.")
		}
	}
	err := stream.Err()
	if context.Cause(ctx) == errIdle {
		return nil, upstream("The provider stopped responding (timed out).")
	}
	if err == nil {
		calls := tracker.finished()
		if !got && len(calls) == 0 {
			return nil, upstream("%s ended without a reply. Check that this provider supports the Responses API.", url)
		}
		return calls, nil // stream closed without a final event; keep what arrived
	}
	err = explain(err, url, true)
	if len(r.Tools) > 0 {
		err = explainTools(err)
	}
	return nil, err
}

// explainTools rewrites a provider's rejection of the tools in a request into
// advice. Providers word it differently ("tools is not supported", "does not
// support function calling", "unknown field tools"), so any 4xx or 501 that
// mentions tools or functions counts.
func explainTools(err error) error {
	var ue *UpstreamError
	if !errors.As(err, &ue) || !ue.rejected {
		return err
	}
	low := strings.ToLower(ue.Msg)
	if !strings.Contains(low, "tool") && !strings.Contains(low, "function") {
		return err
	}
	return upstream("This model or provider doesn't appear to support tools. Turn tools off in the composer "+
		"and send the message again. (%s)", strings.TrimSuffix(ue.Msg, "."))
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
		msg := fmt.Sprintf("The provider returned %d.", apiErr.StatusCode)
		if detail != "" {
			msg = fmt.Sprintf("The provider returned %d: %s", apiErr.StatusCode, detail)
		}
		refused := apiErr.StatusCode == http.StatusNotImplemented ||
			(apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 && apiErr.StatusCode != http.StatusTooManyRequests)
		return &UpstreamError{Msg: msg, rejected: refused}
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
