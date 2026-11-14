package server

import (
	"net/http"
	"regexp"
	"strings"

	"klean/backend/internal/config"
	"klean/backend/internal/store"
	"klean/backend/internal/tools"
)

var skillNamePattern = regexp.MustCompile(config.SkillNamePattern)

// skillBody is a skill in a request body.
type skillBody struct {
	Name         opt[string] `json:"name"`
	Description  opt[string] `json:"description"`
	Instructions opt[string] `json:"instructions"`
	Enabled      opt[bool]   `json:"enabled"`
}

// apply changes the fields of sk that the body carries.
func (b skillBody) apply(sk *store.Skill) {
	if v := b.Name.value(); v != nil {
		sk.Name = strings.TrimSpace(*v)
	}
	if v := b.Description.value(); v != nil {
		sk.Description = strings.TrimSpace(*v)
	}
	if v := b.Instructions.value(); v != nil {
		sk.Instructions = *v
	}
	if v := b.Enabled.value(); v != nil {
		sk.Enabled = *v
	}
}

// checkSkill validates sk against the rules and the skills that exist.
func checkSkill(sk store.Skill, existing []*store.Skill) error {
	if err := checkLen("name", sk.Name, 1, config.SkillNameMaxChars); err != nil {
		return err
	}
	if !skillNamePattern.MatchString(sk.Name) {
		return errorf(http.StatusUnprocessableEntity,
			"name may only contain lowercase letters, digits and hyphens (for example weekly-report).")
	}
	for _, other := range existing {
		if other.ID != sk.ID && strings.EqualFold(other.Name, sk.Name) {
			return errorf(http.StatusUnprocessableEntity, "A skill called %s already exists.", other.Name)
		}
	}
	if err := checkLen("description", sk.Description, 1, config.SkillDescriptionMaxChars); err != nil {
		return err
	}
	if strings.TrimSpace(sk.Instructions) == "" {
		return errorf(http.StatusUnprocessableEntity, "instructions are required.")
	}
	return checkLen("instructions", sk.Instructions, 1, config.SkillInstructionsMaxChars)
}

// checkedSkill applies the body to sk and validates the result.
func (s *Server) checkedSkill(body skillBody, sk store.Skill) (store.Skill, error) {
	body.apply(&sk)
	existing, err := s.db.ListSkills()
	if err != nil {
		return sk, err
	}
	return sk, checkSkill(sk, existing)
}

// enabledSkills lists the skills the model may load, by name.
func (s *Server) enabledSkills() ([]*store.Skill, error) {
	all, err := s.db.ListSkills()
	if err != nil {
		return nil, err
	}
	var out []*store.Skill
	for _, sk := range all {
		if sk.Enabled {
			out = append(out, sk)
		}
	}
	return out, nil
}

// skillInfo is what the model is told about the skills up front.
func skillInfo(skills []*store.Skill) []tools.Skill {
	out := make([]tools.Skill, len(skills))
	for i, sk := range skills {
		out[i] = tools.Skill{Name: sk.Name, Description: sk.Description}
	}
	return out
}

func (s *Server) listSkills(w http.ResponseWriter, r *http.Request) {
	skills, err := s.db.ListSkills()
	reply(w, 200, skills, err)
}

func (s *Server) createSkill(w http.ResponseWriter, r *http.Request) {
	var body skillBody
	sk, err := func() (*store.Skill, error) {
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		sk, err := s.checkedSkill(body, store.Skill{Enabled: true})
		if err != nil {
			return nil, err
		}
		return s.db.CreateSkill(sk)
	}()
	reply(w, http.StatusCreated, sk, err)
}

func (s *Server) updateSkill(w http.ResponseWriter, r *http.Request) {
	var body skillBody
	sk, err := func() (*store.Skill, error) {
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		old, err := s.db.GetSkill(r.PathValue("id"))
		if err != nil || old == nil {
			return nil, orNotFound(err, "Skill not found.")
		}
		sk, err := s.checkedSkill(body, *old)
		if err != nil {
			return nil, err
		}
		return &sk, s.db.SaveSkill(&sk)
	}()
	reply(w, 200, sk, err)
}

func (s *Server) deleteSkill(w http.ResponseWriter, r *http.Request) {
	found, err := s.db.DeleteSkill(r.PathValue("id"))
	if err == nil && !found {
		err = orNotFound(nil, "Skill not found.")
	}
	reply(w, http.StatusNoContent, nil, err)
}
