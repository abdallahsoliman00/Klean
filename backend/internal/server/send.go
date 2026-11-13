package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"klean/backend/internal/config"
	"klean/backend/internal/llm"
	"klean/backend/internal/prompt"
	"klean/backend/internal/store"
	"klean/backend/internal/tools"
)

type sendBody struct {
	Content       string `json:"content"`
	ProviderID    string `json:"providerId"`
	Model         string `json:"model"`
	EditMessageID string `json:"editMessageId"`
	Regenerate    bool   `json:"regenerate"`
	// Tools lists the enabled tool groups. Omitted means all of them; an
	// empty list sends no tools to the provider.
	Tools *[]string `json:"tools"`
	// Answer replies to a pending ask_user call instead of sending a message.
	Answer *struct {
		CallID string `json:"callId"`
		Text   string `json:"text"`
	} `json:"answer"`
}

// answered is the `answered` event: an earlier message's question, now closed.
type answered struct {
	MessageID string     `json:"messageId"`
	Part      store.Part `json:"part"`
}

// prepared is everything needed to stream a reply once the request checks out.
type prepared struct {
	req      llm.Request   // without Turns, which are rebuilt for every round
	history  []prompt.Turn // the whole chat; trimmed to the window per request
	window   int
	reserved int            // tokens the tool definitions take
	groups   []string       // enabled tool groups
	userMsg  *store.Message // nil when regenerating or answering
	answered *answered      // set when the request answered a question
	title    *string        // set when this message named the chat
}

func toolGroups(requested *[]string) ([]string, error) {
	if requested == nil {
		return tools.AllGroups, nil
	}
	for _, g := range *requested {
		if !tools.ValidGroup(g) {
			return nil, errorf(http.StatusUnprocessableEntity, "tools may only contain artifacts, ask and web.")
		}
	}
	return *requested, nil
}

func (s *Server) prepareSend(chatID string, body sendBody) (*prepared, error) {
	if body.ProviderID == "" || body.Model == "" {
		return nil, errorf(http.StatusUnprocessableEntity, "providerId and model are required.")
	}
	groups, err := toolGroups(body.Tools)
	if err != nil {
		return nil, err
	}
	chat, err := s.db.GetChat(chatID)
	if err != nil || chat == nil {
		return nil, orNotFound(err, "Chat not found.")
	}
	provider, err := s.db.GetProvider(body.ProviderID)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, errorf(http.StatusBadRequest, "That provider no longer exists. Pick another model.")
	}

	out := &prepared{groups: groups, window: chat.ContextWindow}
	if body.Answer != nil {
		if out.answered, err = s.answerQuestion(chatID, body.Answer.CallID, body.Answer.Text); err != nil {
			return nil, err
		}
	} else if out.userMsg, err = s.addUserMessage(chatID, body); err != nil {
		return nil, err
	}

	pid, model := &body.ProviderID, &body.Model
	patch := store.ChatPatch{ProviderID: &pid, Model: &model}
	if chat.Title == config.DefaultChatTitle && out.userMsg != nil {
		t := prompt.MakeTitle(out.userMsg.Content)
		out.title, patch.Title = &t, &t
	}
	if chat, err = s.db.UpdateChat(chatID, patch, true); err != nil {
		return nil, err
	}

	history, err := s.db.Messages(chatID)
	if err != nil {
		return nil, err
	}
	// An answer continues from the assistant message that asked; anything
	// else needs a user message to reply to.
	if len(history) == 0 || (out.answered == nil && history[len(history)-1].Role != "user") {
		return nil, errorf(http.StatusConflict, "There's no message to answer.")
	}
	out.history = historyTurns(history, len(groups) > 0)

	defs := tools.Defs(groups)
	if len(defs) > 0 {
		schema, _ := json.Marshal(defs)
		out.reserved = prompt.EstimateTokens(string(schema))
	}
	out.req = llm.Request{
		BaseURL: provider.BaseURL, APIKey: provider.APIKey, Model: body.Model,
		Instructions: withGuidance(chat.SystemPrompt, tools.Guidance(groups)),
		Temperature:  chat.Temperature,
		Tools:        defs,
	}
	return out, nil
}

func withGuidance(systemPrompt, guidance string) string {
	if guidance == "" || systemPrompt == "" {
		return systemPrompt + guidance
	}
	return systemPrompt + "\n\n" + guidance
}

// addUserMessage applies an edit or regenerate, closes a question the user
// chose not to answer, and saves the new message (nil when regenerating).
func (s *Server) addUserMessage(chatID string, body sendBody) (*store.Message, error) {
	content := strings.TrimSpace(body.Content)
	if !body.Regenerate && content == "" {
		return nil, errorf(http.StatusUnprocessableEntity, "Message is empty.")
	}
	if body.EditMessageID != "" {
		found, err := s.db.TruncateFrom(chatID, body.EditMessageID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errorf(http.StatusNotFound, "The message being edited was not found.")
		}
	}
	if body.Regenerate {
		if err := s.db.DropTrailingAssistant(chatID); err != nil {
			return nil, err
		}
	}
	if err := s.skipPendingQuestions(chatID); err != nil {
		return nil, err
	}
	if body.Regenerate {
		return nil, nil
	}
	return s.db.AddMessage(chatID, "user", content, "", false, nil)
}

const skippedOutput = "The user did not answer this question and sent a new message instead."

// answerQuestion closes the pending ask_user call with the user's answer.
func (s *Server) answerQuestion(chatID, callID, text string) (*answered, error) {
	history, err := s.db.Messages(chatID)
	if err != nil {
		return nil, err
	}
	text = strings.TrimSpace(text)
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		for j := range m.Parts {
			p := &m.Parts[j]
			if p.Type != store.PartTool || p.ID != callID || p.Status != store.StatusPending {
				continue
			}
			closeQuestion(p, text)
			if err := s.db.SetMessageParts(m.ID, m.Parts); err != nil {
				return nil, err
			}
			return &answered{MessageID: m.ID, Part: *p}, nil
		}
	}
	return nil, errorf(http.StatusConflict, "That question is no longer waiting for an answer.")
}

// closeQuestion marks a pending ask_user part done. An empty answer means the
// user skipped it.
func closeQuestion(p *store.Part, answer string) {
	p.Status = store.StatusDone
	p.Result, _ = json.Marshal(map[string]string{"answer": answer})
	if p.Output = "The user answered: " + answer; answer == "" {
		p.Output = skippedOutput
	}
}

// skipPendingQuestions closes any question still open: the user moved on
// without answering, and a call must have an output before it can be replayed.
func (s *Server) skipPendingQuestions(chatID string) error {
	history, err := s.db.Messages(chatID)
	if err != nil {
		return err
	}
	for _, m := range history {
		changed := false
		for j := range m.Parts {
			if p := &m.Parts[j]; p.Type == store.PartTool && p.Status == store.StatusPending {
				closeQuestion(p, "")
				changed = true
			}
		}
		if changed {
			if err := s.db.SetMessageParts(m.ID, m.Parts); err != nil {
				return err
			}
		}
	}
	return nil
}

// historyTurns lays the saved messages out as model input. A reply with tool
// calls becomes its text and calls in the order they happened. Without tools
// in the request the calls are left out and only the text is sent, so a chat
// can carry on with a model or provider that has no tool support.
func historyTurns(history []*store.Message, withCalls bool) []prompt.Turn {
	var turns []prompt.Turn
	for _, m := range history {
		if len(m.Parts) == 0 {
			turns = append(turns, prompt.Turn{Role: m.Role, Content: m.Content})
			continue
		}
		for _, p := range m.Parts {
			switch {
			case p.Type == store.PartText:
				turns = append(turns, prompt.Turn{Role: m.Role, Content: p.Text})
			case withCalls:
				turns = append(turns, callTurn(p))
			}
		}
	}
	return turns
}

func callTurn(p store.Part) prompt.Turn {
	output := p.Output
	if output == "" {
		output = "(no result)" // a call is never replayed without an output
	}
	args := string(p.Args)
	if args == "" {
		args = "{}"
	}
	return prompt.Turn{Role: "assistant", Call: &prompt.Call{ID: p.ID, Name: p.Name, Arguments: args, Output: output}}
}

// sendMessage streams Server-Sent Events: `user` (the saved user message) or
// `answered` (the question that was answered), then `token` ({delta}) and the
// tool events (`tool_start`, `tool_run`, `artifact`, `tool_end`) as the reply
// is produced, then `done` ({message, title}) or `error` ({message}).
// Closing the connection stops generation; the partial reply is saved as stopped.
func (s *Server) sendMessage(w http.ResponseWriter, r *http.Request) {
	chatID := r.PathValue("id")
	var body sendBody
	if err := decode(r, &body); err != nil {
		reply(w, 0, nil, err)
		return
	}
	prep, err := s.prepareSend(chatID, body)
	if err != nil {
		reply(w, 0, nil, err)
		return
	}

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	send := func(event string, data any) {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(data)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, bytes.TrimSpace(buf.Bytes()))
		_ = rc.Flush()
	}

	if prep.userMsg != nil {
		send("user", prep.userMsg)
	}
	if prep.answered != nil {
		send("answered", prep.answered)
	}
	t := &turn{s: s, chatID: chatID, groups: prep.groups, send: send, started: map[string]bool{}}
	err = t.run(r.Context(), prep)
	t.flushText()

	if err == nil {
		msg, dbErr := s.db.AddMessage(chatID, "assistant", t.content(), body.Model, false, t.parts)
		if dbErr != nil {
			log.Printf("saving reply: %v", dbErr)
			send("error", map[string]string{"message": "The reply couldn't be saved."})
			return
		}
		send("done", map[string]any{"message": msg, "title": prep.title})
		return
	}

	var ue *llm.UpstreamError
	if errors.As(err, &ue) {
		send("error", map[string]string{"message": ue.Msg})
	} else if r.Context().Err() == nil {
		log.Printf("streaming reply: %v", err)
		send("error", map[string]string{"message": "Something went wrong while streaming the reply."})
	}
	// Stopped by the user, or failed partway: keep the text and the finished
	// tool calls that arrived.
	if len(t.parts) > 0 {
		if _, dbErr := s.db.AddMessage(chatID, "assistant", t.content(), body.Model, true, t.parts); dbErr != nil {
			log.Printf("saving partial reply: %v", dbErr)
		}
	}
}

// turn is one assistant reply being produced, possibly over several requests.
type turn struct {
	s       *Server
	chatID  string
	groups  []string
	send    func(event string, data any)
	parts   []store.Part    // finished parts, in order
	text    strings.Builder // text streamed since the last finished part
	started map[string]bool // calls already announced with tool_start
	asked   bool            // an ask_user call is waiting; the turn ends
}

// flushText closes the text streamed so far into a part and returns it.
func (t *turn) flushText() string {
	text := t.text.String()
	t.text.Reset()
	if text != "" {
		t.parts = append(t.parts, store.Part{Type: store.PartText, Text: text})
	}
	return text
}

// content is the reply's plain text: every text part joined.
func (t *turn) content() string {
	var b strings.Builder
	for _, p := range t.parts {
		if p.Type == store.PartText {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

func (t *turn) token(delta string) {
	t.text.WriteString(delta)
	t.send("token", map[string]string{"delta": delta})
}

func (t *turn) toolStart(id, name string) {
	if !t.started[id] {
		t.started[id] = true
		t.send("tool_start", map[string]string{"id": id, "name": name})
	}
}

// run streams the reply, running tool calls and asking the model again until
// it answers without calling any, asks the user something, or reaches the
// round limit. Whatever was produced is left in t, also when it fails.
func (t *turn) run(ctx context.Context, prep *prepared) error {
	var current []prompt.Turn // this reply so far, which is not in the database yet
	for round := 1; ; round++ {
		req := prep.req
		// Older history gives way as this reply's tool results grow.
		base := prompt.BuildReserving(req.Instructions, prep.history, prep.window, prep.reserved+prompt.Cost(current))
		req.Turns = append(base, current...)

		calls, err := llm.StreamReply(ctx, req, llm.Handler{OnDelta: t.token, OnCallStart: t.toolStart})
		if err != nil {
			return err
		}
		if len(calls) == 0 {
			return nil
		}
		if text := t.flushText(); text != "" {
			current = append(current, prompt.Turn{Role: "assistant", Content: text})
		}
		for _, call := range calls {
			part, err := t.runTool(ctx, call)
			if err != nil {
				return err
			}
			t.parts = append(t.parts, part)
			t.send("tool_end", map[string]any{"part": part})
			current = append(current, callTurn(part))
		}
		if t.asked {
			return nil
		}
		if round >= config.MaxToolRounds {
			t.token(fmt.Sprintf("\n\n*Stopped here: the model was still calling tools after %d rounds.*", config.MaxToolRounds))
			return nil
		}
	}
}
