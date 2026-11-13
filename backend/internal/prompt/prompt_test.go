package prompt

import (
	"strings"
	"testing"

	"klean/backend/internal/config"
)

func TestBuildTrimsToWindow(t *testing.T) {
	var hist []Turn
	for i := 0; i < 20; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		hist = append(hist, Turn{Role: role, Content: strings.Repeat("x", 400)})
	}
	out := Build("sys", hist, 1024) // budget 768 tokens, ~104 per message
	if len(out) < 1 || len(out) >= 20 {
		t.Fatalf("kept %d messages", len(out))
	}
	if out[0].Role != "user" {
		t.Fatal("conversation opens with an assistant reply")
	}
	if out[len(out)-1] != hist[len(hist)-1] {
		t.Fatal("latest message missing")
	}
}

func TestBuildAlwaysSendsLatest(t *testing.T) {
	out := Build("", []Turn{{Role: "user", Content: strings.Repeat("y", 100000)}}, 1024)
	if len(out) != 1 {
		t.Fatalf("got %d messages", len(out))
	}
}

// A tool call and its output travel as one turn, are counted in the budget,
// and a reply cut in half by the budget is dropped back to the next user message.
func TestBuildKeepsCallsWhole(t *testing.T) {
	call := func(out string) Turn {
		return Turn{Role: "assistant", Call: &Call{ID: "c1", Name: "fetch_page", Arguments: `{"url":"x"}`, Output: out}}
	}
	big := strings.Repeat("z", 2400) // ~600 tokens
	hist := []Turn{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "let me look"},
		call(big),
		call(big),
		{Role: "assistant", Content: "here you go"},
		{Role: "user", Content: "thanks, and now?"},
	}
	// Budget 768: the last user message, the closing text and one call fit;
	// the second call does not, so the half-kept reply goes entirely.
	out := Build("", hist, 1024)
	if len(out) != 1 || out[0].Content != "thanks, and now?" {
		t.Fatalf("got %+v", out)
	}
	// With room for everything, nothing is dropped and the order is kept.
	out = Build("", hist, 8192)
	if len(out) != len(hist) || out[2].Call == nil || out[2].Call.Output != big {
		t.Fatalf("got %d turns", len(out))
	}
	// Reserved tokens shrink the budget.
	if out = BuildReserving("", hist, 8192, 6000); len(out) != 1 {
		t.Fatalf("reserved: got %d turns", len(out))
	}
	if Cost(hist) < 1200 {
		t.Fatalf("cost ignores call outputs: %d", Cost(hist))
	}
}

func TestMakeTitle(t *testing.T) {
	if got := MakeTitle("  hello \n there "); got != "hello there" {
		t.Fatalf("%q", got)
	}
	long := MakeTitle(strings.Repeat("é", config.AutoTitleChars+8))
	if []rune(long)[config.AutoTitleChars] != '…' || len([]rune(long)) != config.AutoTitleChars+1 {
		t.Fatalf("%q", long)
	}
}
