package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"klean/backend/internal/llm"
	"klean/backend/internal/prompt"
	"klean/backend/internal/store"
)

// opt records whether a JSON field was present and whether it was null,
// so PATCH bodies can tell "leave alone" from "clear".
type opt[T any] struct {
	Set, Null bool
	V         T
}

func (o *opt[T]) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Null = true
		return nil
	}
	return json.Unmarshal(b, &o.V)
}

// value returns a pointer to the value when present and not null.
func (o opt[T]) value() *T {
	if o.Set && !o.Null {
		return &o.V
	}
	return nil
}

// ---------------- providers ----------------

func (s *Server) listProviders(w http.ResponseWriter, r *http.Request) {
	ps, err := s.db.ListProviders()
	reply(w, 200, ps, err)
}

func (s *Server) createProvider(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    opt[string] `json:"name"`
		BaseURL opt[string] `json:"baseUrl"`
		APIKey  opt[string] `json:"apiKey"`
	}
	p, err := func() (*store.Provider, error) {
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		if body.Name.value() == nil || body.BaseURL.value() == nil {
			return nil, errorf(http.StatusUnprocessableEntity, "name and baseUrl are required.")
		}
		if err := checkLen("name", body.Name.V, 1, 80); err != nil {
			return nil, err
		}
		url, err := cleanURL(body.BaseURL.V)
		if err != nil {
			return nil, err
		}
		return s.db.CreateProvider(strings.TrimSpace(body.Name.V), url, body.APIKey.V)
	}()
	reply(w, http.StatusCreated, p, err)
}

func (s *Server) updateProvider(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    opt[string] `json:"name"`
		BaseURL opt[string] `json:"baseUrl"`
		APIKey  opt[string] `json:"apiKey"` // "" clears the key, omitted or null keeps it
	}
	p, err := func() (*store.Provider, error) {
		id := r.PathValue("id")
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		if p, err := s.db.GetProvider(id); err != nil || p == nil {
			return nil, orNotFound(err, "Provider not found.")
		}
		name := body.Name.value()
		if name != nil {
			if err := checkLen("name", *name, 1, 80); err != nil {
				return nil, err
			}
			*name = strings.TrimSpace(*name)
		}
		url := body.BaseURL.value()
		if url != nil {
			cleaned, err := cleanURL(*url)
			if err != nil {
				return nil, err
			}
			url = &cleaned
		}
		return s.db.UpdateProvider(id, name, url, body.APIKey.value())
	}()
	reply(w, 200, p, err)
}

func (s *Server) deleteProvider(w http.ResponseWriter, r *http.Request) {
	reply(w, http.StatusNoContent, nil, s.db.DeleteProvider(r.PathValue("id")))
}

func (s *Server) providerModels(w http.ResponseWriter, r *http.Request) {
	models, err := func() ([]llm.Model, error) {
		p, err := s.db.GetProvider(r.PathValue("id"))
		if err != nil || p == nil {
			return nil, orNotFound(err, "Provider not found.")
		}
		models, err := llm.ListModels(r.Context(), p.BaseURL, p.APIKey)
		return models, asBadGateway(err)
	}()
	reply(w, 200, models, err)
}

// orNotFound passes a database error through, or reports a 404.
func orNotFound(err error, msg string) error {
	if err != nil {
		return err
	}
	return errorf(http.StatusNotFound, "%s", msg)
}

func asBadGateway(err error) error {
	var ue *llm.UpstreamError
	if errors.As(err, &ue) {
		return errorf(http.StatusBadGateway, "%s", ue.Msg)
	}
	return err
}

// ---------------- chats ----------------

func (s *Server) listChats(w http.ResponseWriter, r *http.Request) {
	cs, err := s.db.ListChats()
	reply(w, 200, cs, err)
}

func (s *Server) createChat(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProviderID    opt[string]  `json:"providerId"`
		Model         opt[string]  `json:"model"`
		Title         opt[string]  `json:"title"`
		SystemPrompt  opt[string]  `json:"systemPrompt"`
		Temperature   opt[float64] `json:"temperature"`
		ContextWindow opt[int]     `json:"contextWindow"`
	}
	c, err := func() (*store.ChatWithMessages, error) {
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		n := store.NewChat{
			ProviderID: body.ProviderID.value(), Model: body.Model.value(),
			Title: body.Title.V, SystemPrompt: body.SystemPrompt.V,
			Temperature: 0.7, ContextWindow: 8192,
		}
		if t := body.Temperature.value(); t != nil {
			n.Temperature = *t
		}
		if cw := body.ContextWindow.value(); cw != nil {
			n.ContextWindow = *cw
		}
		if err := checkTemperature(n.Temperature); err != nil {
			return nil, err
		}
		if err := checkContextWindow(n.ContextWindow); err != nil {
			return nil, err
		}
		return s.db.CreateChat(n)
	}()
	reply(w, http.StatusCreated, c, err)
}

func (s *Server) getChat(w http.ResponseWriter, r *http.Request) {
	c, err := s.db.GetChatWithMessages(r.PathValue("id"))
	if err == nil && c == nil {
		err = orNotFound(nil, "Chat not found.")
	}
	reply(w, 200, c, err)
}

func (s *Server) updateChat(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProviderID    opt[string]  `json:"providerId"` // null clears
		Model         opt[string]  `json:"model"`      // null clears
		Title         opt[string]  `json:"title"`
		SystemPrompt  opt[string]  `json:"systemPrompt"`
		Temperature   opt[float64] `json:"temperature"`
		ContextWindow opt[int]     `json:"contextWindow"`
	}
	c, err := func() (*store.Chat, error) {
		id := r.PathValue("id")
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		if c, err := s.db.GetChat(id); err != nil || c == nil {
			return nil, orNotFound(err, "Chat not found.")
		}
		var p store.ChatPatch
		if body.ProviderID.Set {
			v := body.ProviderID.value()
			p.ProviderID = &v
		}
		if body.Model.Set {
			v := body.Model.value()
			p.Model = &v
		}
		if p.Title = body.Title.value(); p.Title != nil {
			if err := checkLen("title", *p.Title, 1, 200); err != nil {
				return nil, err
			}
		}
		p.SystemPrompt = body.SystemPrompt.value()
		if p.Temperature = body.Temperature.value(); p.Temperature != nil {
			if err := checkTemperature(*p.Temperature); err != nil {
				return nil, err
			}
		}
		if p.ContextWindow = body.ContextWindow.value(); p.ContextWindow != nil {
			if err := checkContextWindow(*p.ContextWindow); err != nil {
				return nil, err
			}
		}
		return s.db.UpdateChat(id, p, false)
	}()
	reply(w, 200, c, err)
}

func (s *Server) deleteChat(w http.ResponseWriter, r *http.Request) {
	found, err := s.db.DeleteChat(r.PathValue("id"))
	if err == nil && !found {
		err = orNotFound(nil, "Chat not found.")
	}
	reply(w, http.StatusNoContent, nil, err)
}

func (s *Server) restoreChat(w http.ResponseWriter, r *http.Request) {
	c, err := s.db.RestoreChat(r.PathValue("id"))
	if err == nil && c == nil {
		err = orNotFound(nil, "Chat not found.")
	}
	reply(w, 200, c, err)
}

// ---------------- sending ----------------

type sendBody struct {
	Content       string `json:"content"`
	ProviderID    string `json:"providerId"`
	Model         string `json:"model"`
	EditMessageID string `json:"editMessageId"`
	Regenerate    bool   `json:"regenerate"`
}

// prepared is everything needed to stream a reply once the request checks out.
type prepared struct {
	req     llm.Request
	userMsg *store.Message // nil when regenerating
	title   *string        // set when this message named the chat
}

func (s *Server) prepareSend(chatID string, body sendBody) (*prepared, error) {
	if body.ProviderID == "" || body.Model == "" {
		return nil, errorf(http.StatusUnprocessableEntity, "providerId and model are required.")
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

	out := &prepared{}
	if !body.Regenerate {
		if out.userMsg, err = s.db.AddMessage(chatID, "user", content, "", false); err != nil {
			return nil, err
		}
	}
	pid, model := &body.ProviderID, &body.Model
	patch := store.ChatPatch{ProviderID: &pid, Model: &model}
	if chat.Title == "New chat" && out.userMsg != nil {
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
	if len(history) == 0 || history[len(history)-1].Role != "user" {
		return nil, errorf(http.StatusConflict, "There's no message to answer.")
	}
	turns := make([]prompt.Turn, len(history))
	for i, m := range history {
		turns[i] = prompt.Turn{Role: m.Role, Content: m.Content}
	}
	out.req = llm.Request{
		BaseURL: provider.BaseURL, APIKey: provider.APIKey, Model: body.Model,
		Instructions: chat.SystemPrompt,
		Turns:        prompt.Build(chat.SystemPrompt, turns, chat.ContextWindow),
		Temperature:  chat.Temperature,
	}
	return out, nil
}

// sendMessage streams Server-Sent Events: `user` (the saved user message),
// `token` ({delta}), then `done` ({message, title}) or `error` ({message}).
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
	var text strings.Builder
	err = llm.StreamReply(r.Context(), prep.req, func(delta string) {
		text.WriteString(delta)
		send("token", map[string]string{"delta": delta})
	})

	if err == nil {
		msg, dbErr := s.db.AddMessage(chatID, "assistant", text.String(), body.Model, false)
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
	// Stopped by the user, or failed partway: keep what arrived.
	if text.Len() > 0 {
		if _, dbErr := s.db.AddMessage(chatID, "assistant", text.String(), body.Model, true); dbErr != nil {
			log.Printf("saving partial reply: %v", dbErr)
		}
	}
}
