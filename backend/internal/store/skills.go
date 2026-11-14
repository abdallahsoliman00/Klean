package store

import "database/sql"

// Skill is a set of instructions the user saved for the model to load when a
// task calls for it. The model is shown the name and description of the
// enabled ones, and asks for the instructions by name.
type Skill struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Instructions string `json:"instructions"`
	Enabled      bool   `json:"enabled"`
}

const skillCols = "id, name, description, instructions, enabled"

func scanSkill(row interface{ Scan(...any) error }) (*Skill, error) {
	var s Skill
	if err := row.Scan(&s.ID, &s.Name, &s.Description, &s.Instructions, &s.Enabled); err != nil {
		return nil, err
	}
	return &s, nil
}

// ListSkills orders by name. Never nil, so it encodes as [].
func (d *DB) ListSkills() ([]*Skill, error) {
	rows, err := d.sql.Query("SELECT " + skillCols + " FROM skills ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Skill{}
	for rows.Next() {
		s, err := scanSkill(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetSkill returns nil, nil when there is no such skill.
func (d *DB) GetSkill(id string) (*Skill, error) {
	s, err := scanSkill(d.sql.QueryRow("SELECT "+skillCols+" FROM skills WHERE id = ?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

// CreateSkill saves s as a new skill; its ID is ignored.
func (d *DB) CreateSkill(s Skill) (*Skill, error) {
	id := newID()
	_, err := d.sql.Exec("INSERT INTO skills ("+skillCols+", created_at) VALUES (?,?,?,?,?,?)",
		id, s.Name, s.Description, s.Instructions, s.Enabled, NowMs())
	if err != nil {
		return nil, err
	}
	return d.GetSkill(id)
}

// SaveSkill writes every field of an existing skill.
func (d *DB) SaveSkill(s *Skill) error {
	_, err := d.sql.Exec("UPDATE skills SET name = ?, description = ?, instructions = ?, enabled = ? WHERE id = ?",
		s.Name, s.Description, s.Instructions, s.Enabled, s.ID)
	return err
}

// DeleteSkill reports whether the skill existed.
func (d *DB) DeleteSkill(id string) (bool, error) {
	res, err := d.sql.Exec("DELETE FROM skills WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
