package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"

	"klean/backend/internal/config"
	"klean/backend/internal/prompt"
)

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
		input = append(input, messageItem(t))
	}
	return input
}

// messageItem is a message as plain text, or as text and image parts when it
// has images.
func messageItem(t prompt.Turn) responses.ResponseInputItemUnionParam {
	role := responses.EasyInputMessageRoleUser
	if t.Role == "assistant" {
		role = responses.EasyInputMessageRoleAssistant
	}
	if len(t.Images) == 0 {
		return responses.ResponseInputItemParamOfMessage(t.Content, role)
	}
	var parts responses.ResponseInputMessageContentListParam
	if t.Content != "" {
		parts = append(parts, responses.ResponseInputContentParamOfInputText(t.Content))
	}
	for _, img := range t.Images {
		p := responses.ResponseInputContentParamOfInputImage(responses.ResponseInputImageDetailAuto)
		p.OfInputImage.ImageURL = openai.String(img.DataURL())
		parts = append(parts, p)
	}
	return responses.ResponseInputItemParamOfMessage(parts, role)
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

// streamResponses is StreamReply over the Responses API.
func streamResponses(ctx context.Context, r Request, h Handler) ([]ToolCall, error) {
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

	ctx, touch, stop := watchIdle(ctx)
	defer stop()

	stream := client.Responses.NewStreaming(ctx, params)
	defer stream.Close()

	tracker := &callTracker{onStart: h.OnCallStart}
	got := false
	for stream.Next() {
		touch()
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
	return endOfStream(ctx, stream.Err(), r, config.ResponsesPath, got, tracker.finished())
}
