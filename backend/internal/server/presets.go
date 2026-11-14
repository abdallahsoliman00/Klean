package server

import (
	"net/http"
	"strings"

	"klean/backend/internal/config"
	"klean/backend/internal/store"
)

// configBody is a chat configuration in a request body, as presets and the
// chat defaults in the settings take it.
type configBody struct {
	ProviderID    opt[string]  `json:"providerId"` // null clears
	Model         opt[string]  `json:"model"`      // null clears
	SystemPrompt  opt[string]  `json:"systemPrompt"`
	Temperature   opt[float64] `json:"temperature"`
	ContextWindow opt[int]     `json:"contextWindow"`
}

// applyConfig changes the fields of c that the body carries and checks them
// the way a chat's are checked.
func (s *Server) applyConfig(body configBody, c *store.ChatConfig) error {
	if body.ProviderID.Set {
		c.ProviderID = body.ProviderID.value()
		if err := s.checkProvider(c.ProviderID); err != nil {
			return err
		}
	}
	if body.Model.Set {
		c.Model = body.Model.value()
	}
	if v := body.SystemPrompt.value(); v != nil {
		c.SystemPrompt = *v
	}
	if v := body.Temperature.value(); v != nil {
		c.Temperature = *v
	}
	if v := body.ContextWindow.value(); v != nil {
		c.ContextWindow = *v
	}
	if err := checkTemperature(c.Temperature); err != nil {
		return err
	}
	return checkContextWindow(c.ContextWindow)
}

// newConfig is the body's configuration on top of the new-chat defaults.
func (s *Server) newConfig(body configBody) (store.ChatConfig, error) {
	c := store.ChatConfig{Temperature: config.DefaultTemperature, ContextWindow: config.DefaultContextWindow}
	return c, s.applyConfig(body, &c)
}

// checkProvider rejects the id of a provider that does not exist.
func (s *Server) checkProvider(id *string) error {
	if id == nil {
		return nil
	}
	p, err := s.db.GetProvider(*id)
	if err != nil {
		return err
	}
	if p == nil {
		return errorf(http.StatusUnprocessableEntity, "That provider doesn't exist.")
	}
	return nil
}

// presetName trims the name and checks its length.
func presetName(name string) (string, error) {
	name = strings.TrimSpace(name)
	return name, checkLen("name", name, 1, config.PresetNameMaxChars)
}

// presetBody is a preset in a request body: its name and its configuration.
type presetBody struct {
	Name opt[string] `json:"name"`
	configBody
}

func (s *Server) listPresets(w http.ResponseWriter, r *http.Request) {
	ps, err := s.db.ListPresets()
	reply(w, 200, ps, err)
}

func (s *Server) createPreset(w http.ResponseWriter, r *http.Request) {
	var body presetBody
	p, err := func() (*store.Preset, error) {
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		name, err := presetName(body.Name.V)
		if err != nil {
			return nil, err
		}
		c, err := s.newConfig(body.configBody)
		if err != nil {
			return nil, err
		}
		return s.db.CreatePreset(name, c)
	}()
	reply(w, http.StatusCreated, p, err)
}

func (s *Server) updatePreset(w http.ResponseWriter, r *http.Request) {
	var body presetBody
	p, err := func() (*store.Preset, error) {
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		p, err := s.db.GetPreset(r.PathValue("id"))
		if err != nil || p == nil {
			return nil, orNotFound(err, "Preset not found.")
		}
		if name := body.Name.value(); name != nil {
			if p.Name, err = presetName(*name); err != nil {
				return nil, err
			}
		}
		if err := s.applyConfig(body.configBody, &p.ChatConfig); err != nil {
			return nil, err
		}
		return p, s.db.SavePreset(p)
	}()
	reply(w, 200, p, err)
}

func (s *Server) deletePreset(w http.ResponseWriter, r *http.Request) {
	found, err := s.db.DeletePreset(r.PathValue("id"))
	if err == nil && !found {
		err = orNotFound(nil, "Preset not found.")
	}
	reply(w, http.StatusNoContent, nil, err)
}
