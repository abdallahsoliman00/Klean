package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"klean/backend/internal/config"
	"klean/backend/internal/llm"
	"klean/backend/internal/store"
	"klean/backend/internal/tools"
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
		if err := checkLen("name", body.Name.V, 1, config.ProviderNameMaxChars); err != nil {
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
			if err := checkLen("name", *name, 1, config.ProviderNameMaxChars); err != nil {
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
			Temperature: config.DefaultTemperature, ContextWindow: config.DefaultContextWindow,
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

// chatDetail is a chat as the thread view loads it. "artifacts" is always
// present, as [] when there are none.
type chatDetail struct {
	*store.ChatWithMessages
	Artifacts []*store.Artifact `json:"artifacts"`
}

func (s *Server) getChat(w http.ResponseWriter, r *http.Request) {
	c, err := func() (*chatDetail, error) {
		c, err := s.db.GetChatWithMessages(r.PathValue("id"))
		if err != nil || c == nil {
			return nil, orNotFound(err, "Chat not found.")
		}
		artifacts, err := s.db.Artifacts(c.ID)
		if err != nil {
			return nil, err
		}
		return &chatDetail{c, artifacts}, nil
	}()
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
			if err := checkLen("title", *p.Title, 1, config.ChatTitleMaxChars); err != nil {
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

// ---------------- artifacts ----------------

func (s *Server) deleteArtifact(w http.ResponseWriter, r *http.Request) {
	found, err := s.db.DeleteArtifact(r.PathValue("id"))
	if err == nil && !found {
		err = orNotFound(nil, "Artifact not found.")
	}
	reply(w, http.StatusNoContent, nil, err)
}

// ---------------- settings ----------------

const (
	settingSearchProvider = "search_provider"
	settingSearchKey      = "search_key"
	settingChatDefaults   = "chat_defaults" // a store.ChatConfig as JSON
)

// settings is what the browser sees. The search key itself never leaves the server.
type settings struct {
	SearchProvider string `json:"searchProvider"`
	HasSearchKey   bool   `json:"hasSearchKey"`
	// ChatDefaults is the configuration used last, which the next new chat
	// starts with. null until one was saved.
	ChatDefaults *store.ChatConfig `json:"chatDefaults"`
}

// chatDefaults returns the saved configuration, or nil when there is none.
// A provider deleted since it was saved is cleared, as it is on chats.
func (s *Server) chatDefaults() (*store.ChatConfig, error) {
	raw, err := s.db.Setting(settingChatDefaults)
	if err != nil || raw == "" {
		return nil, err
	}
	var c store.ChatConfig
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return nil, fmt.Errorf("decoding chat defaults: %w", err)
	}
	if c.ProviderID != nil {
		p, err := s.db.GetProvider(*c.ProviderID)
		if err != nil {
			return nil, err
		}
		if p == nil {
			c.ProviderID = nil
		}
	}
	return &c, nil
}

// saveChatDefaults stores the configuration; null forgets it.
func (s *Server) saveChatDefaults(body opt[configBody]) error {
	if body.Null {
		return s.db.SetSetting(settingChatDefaults, "")
	}
	c, err := s.newConfig(body.V)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.db.SetSetting(settingChatDefaults, string(raw))
}

// searchSettings returns the chosen search provider (DuckDuckGo unless
// another was saved) and its key.
func (s *Server) searchSettings() (provider, key string, err error) {
	if provider, err = s.db.Setting(settingSearchProvider); err != nil {
		return "", "", err
	}
	if !tools.ValidSearchProvider(provider) {
		provider = tools.DuckDuckGo
	}
	key, err = s.db.Setting(settingSearchKey)
	return provider, key, err
}

func (s *Server) readSettings() (*settings, error) {
	provider, key, err := s.searchSettings()
	if err != nil {
		return nil, err
	}
	defaults, err := s.chatDefaults()
	if err != nil {
		return nil, err
	}
	return &settings{SearchProvider: provider, HasSearchKey: key != "", ChatDefaults: defaults}, nil
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	out, err := s.readSettings()
	reply(w, 200, out, err)
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SearchProvider opt[string] `json:"searchProvider"`
		SearchKey      opt[string] `json:"searchKey"` // "" clears the key, omitted or null keeps it
		// ChatDefaults replaces the saved configuration as a whole (fields
		// left out take the new-chat defaults); null forgets it.
		ChatDefaults opt[configBody] `json:"chatDefaults"`
	}
	out, err := func() (*settings, error) {
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		if p := body.SearchProvider.value(); p != nil {
			if !tools.ValidSearchProvider(*p) {
				return nil, errorf(http.StatusUnprocessableEntity, "searchProvider must be duckduckgo, brave or tavily.")
			}
			if err := s.db.SetSetting(settingSearchProvider, *p); err != nil {
				return nil, err
			}
		}
		if k := body.SearchKey.value(); k != nil {
			if err := s.db.SetSetting(settingSearchKey, strings.TrimSpace(*k)); err != nil {
				return nil, err
			}
		}
		if body.ChatDefaults.Set {
			if err := s.saveChatDefaults(body.ChatDefaults); err != nil {
				return nil, err
			}
		}
		return s.readSettings()
	}()
	reply(w, 200, out, err)
}
