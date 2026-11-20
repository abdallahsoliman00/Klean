package tools

import "strings"

// Edit swaps one unique passage of an artifact for another.
type Edit struct {
	OldText string `json:"old_text"`
	NewText string `json:"new_text"`
}

// ApplyEdits applies the edits in order, each to the result of the one
// before. Every old_text must appear exactly once at the time it is applied,
// so an edit never lands on the wrong copy of repeated text. Nothing is
// changed unless every edit applies.
func ApplyEdits(content string, edits []Edit) (string, error) {
	if len(edits) == 0 {
		return "", failf("edits is empty.")
	}
	for i, e := range edits {
		if e.OldText == "" {
			return "", failf("Edit %d: old_text is empty.", i+1)
		}
		switch n := strings.Count(content, e.OldText); n {
		case 1:
			content = strings.Replace(content, e.OldText, e.NewText, 1)
		case 0:
			return "", failf("Edit %d: old_text was not found. It must match the current content exactly, "+
				"including whitespace and indentation. No edits were applied.", i+1)
		default:
			return "", failf("Edit %d: old_text appears %d times. Include more of the surrounding text so it "+
				"matches only one place. No edits were applied.", i+1, n)
		}
	}
	return content, nil
}
