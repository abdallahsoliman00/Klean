package tools

import (
	"strings"
	"testing"
)

func TestApplyEdits(t *testing.T) {
	got, err := ApplyEdits("a b c", []Edit{{"b", "B"}, {"a B", "x"}})
	if err != nil || got != "x c" {
		t.Fatalf("chained edits = %q, %v", got, err)
	}
	for name, c := range map[string]struct {
		edits []Edit
		want  string
	}{
		"none":      {nil, "empty"},
		"blank old": {[]Edit{{"", "x"}}, "empty"},
		"missing":   {[]Edit{{"b", "B"}, {"zzz", "x"}}, "Edit 2: old_text was not found"},
		"repeated":  {[]Edit{{"a", "x"}}, "appears 2 times"},
	} {
		if _, err := ApplyEdits("a b a", c.edits); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want %q", name, err, c.want)
		}
	}
}
