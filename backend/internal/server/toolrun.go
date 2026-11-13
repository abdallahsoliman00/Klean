package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"klean/backend/internal/config"
	"klean/backend/internal/llm"
	"klean/backend/internal/store"
	"klean/backend/internal/tools"
)

// toolFailure is a tool error the model and the user are told about. Any
// other error from a tool ends the turn.
type toolFailure struct{ msg string }

func (e *toolFailure) Error() string { return e.msg }

func failTool(format string, a ...any) error { return &toolFailure{fmt.Sprintf(format, a...)} }

// outcome is what a tool produced: result for the UI, output for the model.
type outcome struct {
	result  any
	output  string
	pending bool // ask_user: no result until the user answers
}

// runTool carries out one call and returns its finished part. A tool that
// fails yields an error part (the model sees the reason and can react); the
// returned error is only for a cancelled request.
func (t *turn) runTool(ctx context.Context, call llm.ToolCall) (store.Part, error) {
	t.toolStart(call.ID, call.Name)
	part := store.Part{Type: store.PartTool, ID: call.ID, Name: call.Name, Args: json.RawMessage("{}")}
	badArgs := !isObject(call.Arguments)
	if !badArgs {
		part.Args = json.RawMessage(call.Arguments)
	}
	t.send("tool_run", map[string]any{"id": call.ID, "name": call.Name, "args": part.Args})

	var out outcome
	var err error
	if badArgs {
		err = failTool("The arguments were not a valid JSON object.")
	} else {
		out, err = t.dispatch(ctx, call.Name, &part)
	}
	if err != nil {
		if ctx.Err() != nil {
			return part, ctx.Err()
		}
		msg := failureMessage(err)
		part.Status, part.Error, part.Output = store.StatusError, msg, "Error: "+msg
		return part, nil
	}
	if out.pending {
		part.Status = store.StatusPending
		return part, nil
	}
	part.Status, part.Output = store.StatusDone, out.output
	part.Result, _ = json.Marshal(out.result)
	return part, nil
}

func isObject(s string) bool {
	var v map[string]any
	return json.Unmarshal([]byte(s), &v) == nil && v != nil
}

func failureMessage(err error) string {
	var tf *toolFailure
	if errors.As(err, &tf) {
		return tf.msg
	}
	var te *tools.Error
	if errors.As(err, &te) {
		return te.Msg
	}
	log.Printf("running tool: %v", err)
	return "The tool failed unexpectedly."
}

// dispatch runs the named tool. It may rewrite part.Args to a tidied form.
func (t *turn) dispatch(ctx context.Context, name string, part *store.Part) (outcome, error) {
	if !t.offered(name) {
		return outcome{}, failTool("There is no tool called %s.", name)
	}
	switch name {
	case tools.CreateArtifact:
		return t.createArtifact(part.Args)
	case tools.UpdateArtifact:
		return t.updateArtifact(part.Args)
	case tools.AskUser:
		return t.askUser(part)
	case tools.WebSearch:
		return t.webSearch(ctx, part.Args)
	default:
		return t.fetchPage(ctx, part.Args)
	}
}

// offered reports whether the tool belongs to a group this request enabled.
func (t *turn) offered(name string) bool {
	for _, def := range tools.Defs(t.groups) {
		if def.Name == name {
			return true
		}
	}
	return false
}

// parseArgs decodes the arguments, explaining a wrong type to the model.
func parseArgs(raw json.RawMessage, dst any) error {
	if err := json.Unmarshal(raw, dst); err != nil {
		return failTool("The arguments don't match the tool's schema: %v", err)
	}
	return nil
}

// ---------------- artifacts ----------------

type artifactResult struct {
	ArtifactID string `json:"artifactId"`
	Title      string `json:"title"`
	Kind       string `json:"kind"`
	Language   string `json:"language"`
	Version    int    `json:"version"`
}

// artifactOutcome announces the artifact and tells the model its id. The
// output is JSON so the id is unambiguous when the model wants to update it.
func (t *turn) artifactOutcome(a *store.Artifact, note string) outcome {
	t.send("artifact", map[string]any{"artifact": a})
	output, _ := json.Marshal(map[string]any{"id": a.ID, "version": a.Version, "note": note})
	return outcome{
		result: artifactResult{ArtifactID: a.ID, Title: a.Title, Kind: a.Kind, Language: a.Language, Version: a.Version},
		output: string(output),
	}
}

func cleanTitle(title string) string {
	title = strings.Join(strings.Fields(title), " ")
	if r := []rune(title); len(r) > config.ArtifactTitleChars {
		title = string(r[:config.ArtifactTitleChars])
	}
	return title
}

func (t *turn) createArtifact(raw json.RawMessage) (outcome, error) {
	var args struct{ Title, Kind, Language, Content string }
	if err := parseArgs(raw, &args); err != nil {
		return outcome{}, err
	}
	kind := strings.ToLower(strings.TrimSpace(args.Kind))
	valid := false
	for _, k := range tools.ArtifactKinds {
		valid = valid || k == kind
	}
	if !valid {
		return outcome{}, failTool("kind must be one of %s.", strings.Join(tools.ArtifactKinds, ", "))
	}
	if strings.TrimSpace(args.Content) == "" {
		return outcome{}, failTool("content is empty.")
	}
	title := cleanTitle(args.Title)
	if title == "" {
		title = "Untitled"
	}
	language := ""
	if kind == "code" {
		language = strings.ToLower(strings.TrimSpace(args.Language))
	}
	a, err := t.s.db.CreateArtifact(t.chatID, title, kind, language, args.Content)
	if err != nil {
		return outcome{}, err
	}
	return t.artifactOutcome(a, "Created. The user sees it in a panel beside the chat; don't repeat its content."), nil
}

func (t *turn) updateArtifact(raw json.RawMessage) (outcome, error) {
	var args struct {
		ID, Content string
		Title       *string
	}
	if err := parseArgs(raw, &args); err != nil {
		return outcome{}, err
	}
	if strings.TrimSpace(args.Content) == "" {
		return outcome{}, failTool("content is empty. Send the complete new content.")
	}
	if args.Title != nil {
		if title := cleanTitle(*args.Title); title == "" {
			args.Title = nil
		} else {
			args.Title = &title
		}
	}
	a, err := t.s.db.UpdateArtifact(t.chatID, strings.TrimSpace(args.ID), args.Content, args.Title)
	if err != nil {
		return outcome{}, err
	}
	if a == nil {
		return outcome{}, failTool("This chat has no artifact with id %q. It may have been deleted; create a new one instead.", args.ID)
	}
	return t.artifactOutcome(a, "Updated. The user sees the new version in the panel."), nil
}

// ---------------- ask_user ----------------

type askOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type askArgs struct {
	Question string      `json:"question"`
	Options  []askOption `json:"options,omitempty"`
	Multiple bool        `json:"multiple,omitempty"`
}

// askUser leaves the call pending: the turn ends and the answer arrives in a
// later request. The arguments are tidied (at most config.MaxAskOptions options, none blank)
// because the UI renders them as they are saved.
func (t *turn) askUser(part *store.Part) (outcome, error) {
	var args askArgs
	if err := parseArgs(part.Args, &args); err != nil {
		return outcome{}, err
	}
	if args.Question = strings.TrimSpace(args.Question); args.Question == "" {
		return outcome{}, failTool("question is empty.")
	}
	if t.asked {
		return outcome{}, failTool("Only one question can be asked at a time. Ask this one after the first is answered.")
	}
	options := args.Options[:0]
	for _, o := range args.Options {
		if o.Label = strings.TrimSpace(o.Label); o.Label != "" && len(options) < config.MaxAskOptions {
			o.Description = strings.TrimSpace(o.Description)
			options = append(options, o)
		}
	}
	args.Options = options
	part.Args, _ = json.Marshal(args)
	t.asked = true
	return outcome{pending: true}, nil
}

// ---------------- web ----------------

func (t *turn) webSearch(ctx context.Context, raw json.RawMessage) (outcome, error) {
	var args struct{ Query string }
	if err := parseArgs(raw, &args); err != nil {
		return outcome{}, err
	}
	provider, key, err := t.s.searchSettings()
	if err != nil {
		return outcome{}, err
	}
	results, err := t.s.web.Search(ctx, provider, key, args.Query)
	if err != nil {
		return outcome{}, err
	}
	query := strings.TrimSpace(args.Query)
	return outcome{
		result: map[string]any{"query": query, "results": results},
		output: tools.FormatResults(query, results),
	}, nil
}

func (t *turn) fetchPage(ctx context.Context, raw json.RawMessage) (outcome, error) {
	var args struct{ URL string }
	if err := parseArgs(raw, &args); err != nil {
		return outcome{}, err
	}
	page, err := t.s.web.Fetch(ctx, args.URL)
	if err != nil {
		return outcome{}, err
	}
	return outcome{
		result: map[string]any{"url": page.URL, "title": page.Title, "chars": page.Chars},
		output: tools.FormatPage(page),
	}, nil
}
