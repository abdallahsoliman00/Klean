// Package prompt decides which messages go into each request so the chat fits
// the model's context window.
package prompt

import (
	"strings"
	"unicode/utf8"

	"klean/backend/internal/config"
)

// Turn is one item sent to the model: a message, or (when Call is set) a tool
// call the assistant made together with what it returned. Keeping a call and
// its output in one Turn means trimming can never separate them.
type Turn struct {
	Role    string // "user" or "assistant"
	Content string
	Call    *Call
}

// Call is a finished tool call.
type Call struct {
	ID        string
	Name      string
	Arguments string // JSON object
	Output    string // what the model was told the call returned
}

// EstimateTokens is rough and model-agnostic: config.CharsPerToken characters
// per token, rounded up.
func EstimateTokens(text string) int {
	return (utf8.RuneCountInString(text) + config.CharsPerToken - 1) / config.CharsPerToken
}

// Cost estimates the tokens the turns take in a request.
func Cost(turns []Turn) int {
	n := 0
	for _, t := range turns {
		n += cost(t)
	}
	return n
}

func cost(t Turn) int {
	if t.Call != nil {
		// Sent as two items: the call and its output.
		return EstimateTokens(t.Call.Name) + EstimateTokens(t.Call.Arguments) +
			EstimateTokens(t.Call.Output) + 2*config.PerMessageOverhead
	}
	return EstimateTokens(t.Content) + config.PerMessageOverhead
}

// Build picks the newest messages until the budget is used. The system prompt
// is always counted (it is sent separately as instructions), and the latest
// message is always sent even if it alone is over budget.
func Build(systemPrompt string, history []Turn, contextWindow int) []Turn {
	return BuildReserving(systemPrompt, history, contextWindow, 0)
}

// BuildReserving is Build with reserved tokens set aside for what else goes
// into the request: tool definitions, and the tool calls of the reply being
// written.
func BuildReserving(systemPrompt string, history []Turn, contextWindow, reserved int) []Turn {
	budget := int(float64(contextWindow) * (1 - config.ReplyShare))
	used := reserved
	if systemPrompt != "" {
		used += EstimateTokens(systemPrompt)
	}
	var picked []Turn
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		if m.Call == nil && m.Content == "" {
			continue
		}
		c := cost(m)
		if len(picked) > 0 && used+c > budget {
			break
		}
		picked = append(picked, m)
		used += c
	}
	// reverse into chronological order
	for i, j := 0, len(picked)-1; i < j; i, j = i+1, j-1 {
		picked[i], picked[j] = picked[j], picked[i]
	}
	// A conversation shouldn't open with an orphaned assistant reply, or with
	// the tail end of one (its tool calls count as assistant turns).
	for len(picked) > 1 && picked[0].Role == "assistant" {
		picked = picked[1:]
	}
	return picked
}

// MakeTitle turns the first message into a short chat title.
func MakeTitle(text string) string {
	t := strings.Join(strings.Fields(text), " ")
	r := []rune(t)
	if len(r) <= config.AutoTitleChars {
		return t
	}
	return strings.TrimRight(string(r[:config.AutoTitleChars]), " \t\n") + "…"
}
