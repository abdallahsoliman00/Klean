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
	if out[len(out)-1].Content != hist[len(hist)-1].Content {
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

func TestCleanTitle(t *testing.T) {
	cases := map[string]string{
		"\"Planning a trip to Rome.\"\n": "Planning a trip to Rome",
		"## **Sorting in Go**":           "Sorting in Go",
		"  two\n lines?! ":               "two lines",
		"Learning C#":                    "Learning C#",
		" \"\" ":                         "",
	}
	for raw, want := range cases {
		if got := CleanTitle(raw); got != want {
			t.Errorf("CleanTitle(%q) = %q, want %q", raw, got, want)
		}
	}
	long := CleanTitle(strings.Repeat("é", config.ChatTitleMaxChars+8))
	if len([]rune(long)) != config.ChatTitleMaxChars {
		t.Fatalf("long title has %d characters", len([]rune(long)))
	}
}

func TestTitleTurns(t *testing.T) {
	turns := TitleTurns(strings.Repeat("u", config.TitleInputChars+5), "the reply")
	if len(turns) != 1 || turns[0].Role != "user" {
		t.Fatalf("turns: %+v", turns)
	}
	c := turns[0].Content
	if strings.Count(c, "u") != config.TitleInputChars || !strings.HasSuffix(c, "Assistant:\nthe reply") {
		t.Fatalf("content: %q", c)
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

// An image counts as a fixed number of tokens, and a message that is only
// images is still sent.
func TestImagesAreCounted(t *testing.T) {
	img := Image{MediaType: "image/png", Data: "AAAA"}
	if img.DataURL() != "data:image/png;base64,AAAA" {
		t.Fatalf("data URL: %s", img.DataURL())
	}
	text := []Turn{{Role: "user", Content: "look"}}
	with := []Turn{{Role: "user", Content: "look", Images: []Image{img, img}}}
	if got := Cost(with) - Cost(text); got != 2*config.ImageTokens {
		t.Fatalf("two images cost %d tokens", got)
	}
	hist := []Turn{
		{Role: "user", Images: []Image{img}},
		{Role: "assistant", Content: "a cat"},
		{Role: "user", Content: "and this?", Images: []Image{img}},
	}
	if out := Build("", hist, 1_000_000); len(out) != 3 || len(out[0].Images) != 1 {
		t.Fatalf("image-only turn dropped: %+v", out)
	}
	// A window with room for one image keeps the latest turn and drops the older image.
	budget := float64(config.ImageTokens + 100)
	window := int(budget / (1 - config.ReplyShare))
	if out := Build("", hist, window); len(out) != 1 || out[0].Content != "and this?" {
		t.Fatalf("got %d turns", len(out))
	}
}
