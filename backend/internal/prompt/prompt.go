// Package prompt decides which messages go into each request so the chat fits
// the model's context window.
package prompt

import (
	"strings"
	"unicode/utf8"
)

const (
	replyShare         = 0.25 // keep a quarter of the window free for the model's answer
	perMessageOverhead = 4
)

// Turn is one message sent to the model.
type Turn struct {
	Role    string // "user" or "assistant"
	Content string
}

// EstimateTokens is rough and model-agnostic: about four characters per token.
func EstimateTokens(text string) int {
	return (utf8.RuneCountInString(text) + 3) / 4
}

// Build picks the newest messages until the budget is used. The system prompt
// is always counted (it is sent separately as instructions), and the latest
// message is always sent even if it alone is over budget.
func Build(systemPrompt string, history []Turn, contextWindow int) []Turn {
	budget := int(float64(contextWindow) * (1 - replyShare))
	used := 0
	if systemPrompt != "" {
		used = EstimateTokens(systemPrompt)
	}
	var picked []Turn
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		if m.Content == "" {
			continue
		}
		cost := EstimateTokens(m.Content) + perMessageOverhead
		if len(picked) > 0 && used+cost > budget {
			break
		}
		picked = append(picked, m)
		used += cost
	}
	// reverse into chronological order
	for i, j := 0, len(picked)-1; i < j; i, j = i+1, j-1 {
		picked[i], picked[j] = picked[j], picked[i]
	}
	// A conversation shouldn't open with an orphaned assistant reply.
	for len(picked) > 1 && picked[0].Role == "assistant" {
		picked = picked[1:]
	}
	return picked
}

// MakeTitle turns the first message into a short chat title.
func MakeTitle(text string) string {
	const limit = 42
	t := strings.Join(strings.Fields(text), " ")
	r := []rune(t)
	if len(r) <= limit {
		return t
	}
	return strings.TrimRight(string(r[:limit]), " \t\n") + "…"
}
