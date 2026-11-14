package llm

import "klean/backend/internal/prompt"

type Model struct {
	ID      string `json:"id"`
	OwnedBy string `json:"ownedBy"`
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
