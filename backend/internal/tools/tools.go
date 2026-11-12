// Package tools defines the functions the model can call and implements the
// ones that reach the web. Artifacts and questions to the user are carried out
// by the server, which owns the database and the event stream.
package tools

import (
	"strings"

	"klean/backend/internal/llm"
)

// Tool names, as the model sees them.
const (
	CreateArtifact = "create_artifact"
	UpdateArtifact = "update_artifact"
	AskUser        = "ask_user"
	WebSearch      = "web_search"
	FetchPage      = "fetch_page"
)

// Groups are what the composer switches on and off.
const (
	GroupArtifacts = "artifacts"
	GroupAsk       = "ask"
	GroupWeb       = "web"
)

// AllGroups is the default when a request doesn't say which groups it wants.
var AllGroups = []string{GroupArtifacts, GroupAsk, GroupWeb}

// ArtifactKinds are the values create_artifact accepts for kind.
var ArtifactKinds = []string{"html", "svg", "markdown", "code"}

func ValidGroup(g string) bool {
	return g == GroupArtifacts || g == GroupAsk || g == GroupWeb
}

type obj = map[string]any

func str(desc string) obj { return obj{"type": "string", "description": desc} }

func object(required []string, props obj) obj {
	return obj{"type": "object", "properties": props, "required": required, "additionalProperties": false}
}

var defs = map[string][]llm.Tool{
	GroupArtifacts: {
		{
			Name: CreateArtifact,
			Description: "Create an artifact: a standalone piece of content shown to the user in a side panel next to the chat, " +
				"where they can preview, copy and download it. Use it for substantial, self-contained work the user will keep " +
				"or reuse: a web page, an SVG image, a document, or a complete source file. Do not use it for short snippets " +
				"or for ordinary answers. The result gives the artifact's id, which update_artifact needs.",
			Parameters: object([]string{"title", "kind", "content"}, obj{
				"title": str("Short name shown above the artifact. For code, use the file name (for example main.py)."),
				"kind": obj{"type": "string", "enum": ArtifactKinds, "description": "html: a complete HTML page, rendered in a preview " +
					"(inline its CSS and JavaScript). svg: one <svg> element, rendered as an image. markdown: a document. " +
					"code: a source file or other plain text, shown with syntax highlighting."},
				"language": str("For kind=code only: the language for highlighting, lower case (python, go, typescript, bash...)."),
				"content":  str("The full content of the artifact."),
			}),
		},
		{
			Name: UpdateArtifact,
			Description: "Replace the content of an artifact created earlier in this chat. Send the complete new content, " +
				"not a diff. The artifact keeps its id and its version goes up by one.",
			Parameters: object([]string{"id", "content"}, obj{
				"id":      str("The id that create_artifact returned."),
				"content": str("The full new content, replacing the old."),
				"title":   str("A new title. Leave out to keep the current one."),
			}),
		},
	},
	GroupAsk: {
		{
			Name: AskUser,
			Description: "Ask the user one question and wait for the answer. The question is shown with the options as buttons " +
				"and a free-text field, and your turn ends until they reply. Use it only when you cannot continue without a " +
				"decision that is the user's to make; otherwise make a reasonable choice and say what you chose.",
			Parameters: object([]string{"question"}, obj{
				"question": str("The question, in one or two sentences."),
				"options": obj{
					"type": "array", "maxItems": 4,
					"description": "Up to four suggested answers. Leave out for an open question.",
					"items": object([]string{"label"}, obj{
						"label":       str("The answer, a few words."),
						"description": str("One line on what choosing it means."),
					}),
				},
				"multiple": obj{"type": "boolean", "description": "True if the user may pick more than one option."},
			}),
		},
	},
	GroupWeb: {
		{
			Name: WebSearch,
			Description: "Search the web. Returns the top results as title, URL and a short snippet. Use it for current events " +
				"and for facts you are unsure of. Snippets are brief; call fetch_page on a result to read it.",
			Parameters: object([]string{"query"}, obj{
				"query": str("The search query, as you would type it into a search engine."),
			}),
		},
		{
			Name: FetchPage,
			Description: "Download a public web page and return its readable text (truncated when long). " +
				"Only http and https URLs on the public internet work.",
			Parameters: object([]string{"url"}, obj{
				"url": str("The full URL, starting with http:// or https://."),
			}),
		},
	},
}

// Defs returns the tools of the enabled groups, in a fixed order.
func Defs(groups []string) []llm.Tool {
	var out []llm.Tool
	for _, g := range AllGroups {
		if has(groups, g) {
			out = append(out, defs[g]...)
		}
	}
	return out
}

func has(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

var guidance = map[string]string{
	GroupArtifacts: "Put substantial standalone work (a web page, an SVG, a document, a complete file or script) in an artifact " +
		"with create_artifact, and change it with update_artifact rather than creating another. Keep short examples, " +
		"one-liners and explanations inline in your reply. After creating or updating an artifact, describe it in a " +
		"sentence or two; don't repeat its content.",
	GroupAsk: "Call ask_user only when you are genuinely blocked on a choice that is the user's to make. " +
		"For anything else, pick a sensible default and mention it.",
	GroupWeb: "Use web_search for recent or uncertain facts and fetch_page to read a result. " +
		"When your answer relies on web results, cite the URLs you used.",
}

// Guidance is appended to the instructions when tools are offered. It returns
// "" when no group is enabled.
func Guidance(groups []string) string {
	var lines []string
	for _, g := range AllGroups {
		if has(groups, g) {
			lines = append(lines, guidance[g])
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "You can call tools. " + strings.Join(lines, " ")
}
