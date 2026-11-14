package store

import "database/sql"

// ChatConfig is the configuration a chat runs with. Presets save one under a
// name, and the settings keep the one used last for the next new chat.
type ChatConfig struct {
	ProviderID    *string `json:"providerId"`
	Model         *string `json:"model"`
	SystemPrompt  string  `json:"systemPrompt"`
	Temperature   float64 `json:"temperature"`
	ContextWindow int     `json:"contextWindow"`
}

// Preset is a saved, named chat configuration.
type Preset struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	ChatConfig
}

const presetCols = "id, name, provider_id, model, system_prompt, temperature, context_window"

func scanPreset(row interface{ Scan(...any) error }) (*Preset, error) {
	var p Preset
	var pid, model sql.NullString
	if err := row.Scan(&p.ID, &p.Name, &pid, &model, &p.SystemPrompt, &p.Temperature, &p.ContextWindow); err != nil {
		return nil, err
	}
	if pid.Valid {
		p.ProviderID = &pid.String
	}
	if model.Valid {
		p.Model = &model.String
	}
	return &p, nil
}

// ListPresets orders by name. Never nil, so it encodes as [].
func (d *DB) ListPresets() ([]*Preset, error) {
	rows, err := d.sql.Query("SELECT " + presetCols + " FROM presets ORDER BY name COLLATE NOCASE, created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Preset{}
	for rows.Next() {
		p, err := scanPreset(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetPreset returns nil, nil when there is no such preset.
func (d *DB) GetPreset(id string) (*Preset, error) {
	p, err := scanPreset(d.sql.QueryRow("SELECT "+presetCols+" FROM presets WHERE id = ?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (d *DB) CreatePreset(name string, c ChatConfig) (*Preset, error) {
	id := newID()
	_, err := d.sql.Exec("INSERT INTO presets ("+presetCols+", created_at) VALUES (?,?,?,?,?,?,?,?)",
		id, name, c.ProviderID, c.Model, c.SystemPrompt, c.Temperature, c.ContextWindow, NowMs())
	if err != nil {
		return nil, err
	}
	return d.GetPreset(id)
}

// SavePreset writes every field of an existing preset.
func (d *DB) SavePreset(p *Preset) error {
	_, err := d.sql.Exec(`UPDATE presets SET name = ?, provider_id = ?, model = ?, system_prompt = ?,
		temperature = ?, context_window = ? WHERE id = ?`,
		p.Name, p.ProviderID, p.Model, p.SystemPrompt, p.Temperature, p.ContextWindow, p.ID)
	return err
}

// DeletePreset reports whether the preset existed.
func (d *DB) DeletePreset(id string) (bool, error) {
	res, err := d.sql.Exec("DELETE FROM presets WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
