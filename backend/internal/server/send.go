package server

import (
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
	Content    string `json:"content"`
	ProviderID string `json:"providerId"`
	Model      string `json:"model"`
	// Images are attached to a new message. An edit keeps the images the
	// message had and ignores these.
	Images        []imageBody `json:"images"`
	EditMessageID string      `json:"editMessageId"`
	Regenerate    bool        `json:"regenerate"`
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
	skills   []*store.Skill // the enabled skills, which use_skill may load
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

	skills, err := s.enabledSkills()
	if err != nil {
		return nil, err
	}
	// use_skill doesn't belong to a group: it is offered whenever there are skills.
	defs := tools.Defs(groups)
	if len(skills) > 0 {
		defs = append(defs, tools.SkillDef)
	}

	out := &prepared{skills: skills, window: chat.ContextWindow}
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
	imageData, err := s.db.ChatImageData(chatID)
	if err != nil {
		return nil, err
	}
	out.history = historyTurns(history, len(defs) > 0, imageData)

	if len(defs) > 0 {
		schema, _ := json.Marshal(defs)
		out.reserved = prompt.EstimateTokens(string(schema))
	}
	out.req = llm.Request{
		BaseURL: provider.BaseURL, APIKey: provider.APIKey, Model: body.Model,
		Instructions: instructions(body.Model, chat.SystemPrompt, tools.Guidance(groups), tools.SkillGuidance(skillInfo(skills))),
		Temperature:  chat.Temperature,
		Tools:        defs,
	}
	return out, nil
}

// instructions is the system prompt the model gets: Klean's introduction,
// the user's own prompt, then the guidance for tools and skills.
func instructions(model, systemPrompt, toolGuidance, skillGuidance string) string {
	out := fmt.Sprintf(config.Intro, model)
	for _, section := range []string{systemPrompt, toolGuidance, skillGuidance} {
		out = withGuidance(out, section)
	}
	return out
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
	if body.Regenerate {
		if err := s.db.DropTrailingAssistant(chatID); err != nil {
			return nil, err
		}
		return nil, s.skipPendingQuestions(chatID)
	}
	content := strings.TrimSpace(body.Content)
	if body.EditMessageID != "" {
		return s.editMessage(chatID, body.EditMessageID, content)
	}
	images, err := parseImages(body.Images)
	if err != nil {
		return nil, err
	}
	if content == "" && len(images) == 0 {
		return nil, errEmptyMessage
	}
	if err := s.skipPendingQuestions(chatID); err != nil {
		return nil, err
	}
	return s.db.AddUserMessage(chatID, content, images)
}

var errEmptyMessage = errorf(http.StatusUnprocessableEntity, "Message is empty.")

// editMessage replaces a message and everything after it with new text. The
// message keeps its images, so the text may be empty when it has some.
func (s *Server) editMessage(chatID, messageID, content string) (*store.Message, error) {
	kept, err := s.db.MessageImages(messageID)
	if err != nil {
		return nil, err
	}
	if content == "" && len(kept) == 0 {
		return nil, errEmptyMessage
	}
	msg, err := s.db.ReplaceFrom(chatID, messageID, content)
	if err != nil || msg == nil {
		return nil, orNotFound(err, "The message being edited was not found.")
	}
	return msg, s.skipPendingQuestions(chatID)
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
// imageData holds the bytes of the chat's images by image id.
func historyTurns(history []*store.Message, withCalls bool, imageData map[string][]byte) []prompt.Turn {
	var turns []prompt.Turn
	for _, m := range history {
		if len(m.Parts) == 0 {
			turns = append(turns, prompt.Turn{Role: m.Role, Content: m.Content, Images: turnImages(m.Images, imageData)})
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

// sendMessage starts a reply and streams it as Server-Sent Events: `user`
// (the saved user message) or `answered` (the question that was answered),
// then `token` ({delta}) and the tool events (`tool_start`, `tool_run`,
// `artifact`, `tool_end`) as the reply is produced, then `done` ({message,
// title}), `error` ({message}) or `stopped` ({message}, null when nothing had
// arrived). After the `done` of the message that named the chat, `title`
// ({title}) follows when the model could summarise the conversation into a
// better title.
// The reply is written whether or not anyone is listening: closing the
// connection leaves it running, and streamReply attaches to it again. A chat
// that already has a reply running answers 409.
func (s *Server) sendMessage(w http.ResponseWriter, r *http.Request) {
	chatID := r.PathValue("id")
	var body sendBody
	if err := decodeLimit(r, &body, config.MaxMessageBody); err != nil {
		reply(w, 0, nil, err)
		return
	}
	// The chat is claimed first: preparing already changes its messages.
	rn, err := s.startRun(chatID)
	if err != nil {
		reply(w, 0, nil, err)
		return
	}
	prep, err := s.prepareSend(chatID, body)
	if err != nil {
		s.markSaved(chatID, rn)
		s.endRun(rn)
		reply(w, 0, nil, err)
		return
	}
	if prep.userMsg != nil {
		rn.send("user", prep.userMsg)
	}
	if prep.answered != nil {
		rn.send("answered", prep.answered)
	}
	go s.generate(rn, chatID, body.Model, prep)
	rn.streamTo(w, r, nil)
}

// generate writes the reply and saves it: whole, or as far as it got when it
// was stopped or failed (the text and the finished tool calls).
func (s *Server) generate(rn *run, chatID, model string, prep *prepared) {
	defer s.endRun(rn)
	t := &turn{s: s, chatID: chatID, tools: prep.req.Tools, skills: prep.skills, send: rn.send, started: map[string]bool{}}
	err := t.run(rn.ctx, prep)
	t.flushText()

	var msg *store.Message
	var dbErr error
	if err == nil || len(t.parts) > 0 {
		if msg, dbErr = s.db.AddMessage(chatID, "assistant", t.content(), model, err != nil, t.parts); dbErr != nil {
			log.Printf("saving reply: %v", dbErr)
		}
	}
	s.markSaved(chatID, rn)

	var ue *llm.UpstreamError
	switch {
	case err == nil && dbErr != nil:
		rn.send("error", map[string]string{"message": "The reply couldn't be saved."})
	case err == nil:
		rn.send("done", map[string]any{"message": msg, "title": prep.title})
		if prep.title != nil {
			if title := s.summaryTitle(rn.ctx, chatID, prep, msg.Content); title != "" {
				rn.send("title", map[string]string{"title": title})
			}
		}
	case rn.ctx.Err() != nil: // stopped, the chat was deleted, or Klean is closing
		rn.send("stopped", map[string]any{"message": msg})
	case errors.As(err, &ue):
		rn.send("error", map[string]string{"message": ue.Msg})
	default:
		log.Printf("streaming reply: %v", err)
		rn.send("error", map[string]string{"message": "Something went wrong while streaming the reply."})
	}
}

// turn is one assistant reply being produced, possibly over several requests.
type turn struct {
	s       *Server
	chatID  string
	tools   []llm.Tool     // the tools this request offered
	skills  []*store.Skill // the enabled skills
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
// round limit, where it is asked for a final answer without tools. Whatever was produced is left in t, also when it fails.
func (t *turn) run(ctx context.Context, prep *prepared) error {
	var current []prompt.Turn // this reply so far, which is not in the database yet
	maxRounds := config.MaxToolRounds
	for round := 1; ; round++ {
		req := prep.req
		// Out of rounds: ask once more without tools, so the model reports what
		// it found instead of the reply ending on a tool call.
		final := round > maxRounds
		if final {
			req.Tools = nil
			current = append(current, prompt.Turn{Role: "user", Content: config.ToolLimitPrompt})
		}
		// Older history gives way as this reply's tool results grow.
		base := prompt.BuildReserving(req.Instructions, prep.history, prep.window, prep.reserved+prompt.Cost(current))
		req.Turns = append(base, current...)

		calls, err := llm.StreamReply(ctx, req, llm.Handler{OnDelta: t.token, OnCallStart: t.toolStart})
		if err != nil {
			return err
		}
		if len(calls) == 0 || final {
			return nil
		}
		if anyWebCall(calls) {
			maxRounds = config.MaxWebSearchToolRounds
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
	}
}

// anyWebCall reports whether a round called a tool that reaches the web.
// Such a reply gets the larger round limit: research takes several searches
// and page reads before the model can answer.
func anyWebCall(calls []llm.ToolCall) bool {
	for _, c := range calls {
		if tools.IsWeb(c.Name) {
			return true
		}
	}
	return false
}
