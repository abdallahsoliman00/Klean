package prompt

import (
	"strings"
	"testing"
)

func TestBuildTrimsToWindow(t *testing.T) {
	var hist []Turn
	for i := 0; i < 20; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		hist = append(hist, Turn{role, strings.Repeat("x", 400)})
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
	out := Build("", []Turn{{"user", strings.Repeat("y", 100000)}}, 1024)
	if len(out) != 1 {
		t.Fatalf("got %d messages", len(out))
	}
}

func TestMakeTitle(t *testing.T) {
	if got := MakeTitle("  hello \n there "); got != "hello there" {
		t.Fatalf("%q", got)
	}
	long := MakeTitle(strings.Repeat("é", 50))
	if []rune(long)[42] != '…' || len([]rune(long)) != 43 {
		t.Fatalf("%q", long)
	}
}
