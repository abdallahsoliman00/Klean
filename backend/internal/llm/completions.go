package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"

	"klean/backend/internal/config"
	"klean/backend/internal/prompt"
)

// buildMessages turns the instructions and the history into Chat Completions
// messages. An earlier tool call is replayed as an assistant message carrying
// the call, followed by a tool message with its output.
func buildMessages(instructions string, turns []prompt.Turn) []openai.ChatCompletionMessageParamUnion {
	msgs := make([]openai.ChatCompletionMessageParamUnion, 0, len(turns)+1)
	if instructions != "" {
		msgs = append(msgs, openai.SystemMessage(instructions))
	}
	for _, t := range turns {
		switch c := t.Call; {
		case c != nil:
			call := openai.ChatCompletionMessageFunctionToolCallParam{ID: c.ID}
			call.Function.Name, call.Function.Arguments = c.Name, c.Arguments
			asked := openai.ChatCompletionAssistantMessageParam{
				ToolCalls: []openai.ChatCompletionMessageToolCallUnionParam{{OfFunction: &call}},
			}
			msgs = append(msgs, openai.ChatCompletionMessageParamUnion{OfAssistant: &asked}, openai.ToolMessage(c.Output, c.ID))
		case t.Role == "assistant":
			msgs = append(msgs, openai.AssistantMessage(t.Content))
		case len(t.Images) == 0:
			msgs = append(msgs, openai.UserMessage(t.Content))
		default:
			msgs = append(msgs, openai.UserMessage(imageParts(t)))
		}
	}
	return msgs
}

// imageParts is a user message as text and image_url parts; the images
// travel as data: URLs.
func imageParts(t prompt.Turn) []openai.ChatCompletionContentPartUnionParam {
	var parts []openai.ChatCompletionContentPartUnionParam
	if t.Content != "" {
		parts = append(parts, openai.TextContentPart(t.Content))
	}
	for _, img := range t.Images {
		parts = append(parts, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{URL: img.DataURL()}))
	}
	return parts
}

func buildFunctionTools(tools []Tool) []openai.ChatCompletionToolUnionParam {
	out := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, t := range tools {
		// Not strict, for the same reasons as on the Responses path.
		out = append(out, openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name: t.Name, Description: openai.String(t.Description), Parameters: t.Parameters,
		}))
	}
	return out
}

// chunkCalls pieces tool calls together from stream chunks: each chunk
// carries a call's position and the next piece of its arguments, and the
// first one its id and name.
type chunkCalls struct {
	calls   []*ToolCall
	onStart func(id, name string)
}

func (c *chunkCalls) add(d openai.ChatCompletionChunkChoiceDeltaToolCall) {
	if d.Index < 0 {
		return
	}
	for int64(len(c.calls)) <= d.Index {
		c.calls = append(c.calls, &ToolCall{})
	}
	call := c.calls[d.Index]
	announce := call.Name == "" && d.Function.Name != ""
	if announce {
		call.Name = d.Function.Name
		// Some servers leave the id out; the model only needs it to be consistent.
		call.ID = firstNonEmpty(call.ID, d.ID, fmt.Sprintf("call_%d", d.Index+1))
	}
	call.Arguments += d.Function.Arguments
	if announce && c.onStart != nil {
		c.onStart(call.ID, call.Name)
	}
}

// finished returns the calls in the order the model made them.
func (c *chunkCalls) finished() []ToolCall {
	var out []ToolCall
	for _, call := range c.calls {
		if call.Name != "" {
			out = append(out, *call)
		}
	}
	return out
}

// streamCompletions is StreamReply over the Chat Completions API.
func streamCompletions(ctx context.Context, r Request, h Handler) ([]ToolCall, error) {
	client := newClient(r.BaseURL, r.APIKey)

	params := openai.ChatCompletionNewParams{
		Model:       r.Model,
		Messages:    buildMessages(r.Instructions, r.Turns),
		Temperature: openai.Float(r.Temperature),
	}
	if len(r.Tools) > 0 {
		params.Tools = buildFunctionTools(r.Tools)
	}

	ctx, touch, stop := watchIdle(ctx)
	defer stop()

	stream := client.Chat.Completions.NewStreaming(ctx, params)
	defer stream.Close()

	calls := &chunkCalls{onStart: h.OnCallStart}
	got := false
	for stream.Next() {
		touch()
		for _, choice := range stream.Current().Choices {
			if text := choice.Delta.Content + choice.Delta.Refusal; text != "" {
				got = true
				if h.OnDelta != nil {
					h.OnDelta(text)
				}
			}
			for _, d := range choice.Delta.ToolCalls {
				calls.add(d)
			}
		}
	}
	return endOfStream(ctx, stream.Err(), r, config.CompletionsPath, got, calls.finished())
}
