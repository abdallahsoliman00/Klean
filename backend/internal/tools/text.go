package tools

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"klean/backend/internal/config"
)

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasAttr(n *html.Node, key string) bool {
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}

func hasClass(n *html.Node, class string) bool {
	for _, c := range strings.Fields(attr(n, "class")) {
		if c == class {
			return true
		}
	}
	return false
}

// nodeText is the text inside a node, tags removed.
func nodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// collapse joins runs of whitespace into single spaces.
func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// plainText strips the tags and entities from a fragment of HTML.
func plainText(fragment string) string {
	doc, err := html.Parse(strings.NewReader(fragment))
	if err != nil {
		return collapse(fragment)
	}
	return collapse(nodeText(doc))
}

func clip(s string, max int) string {
	if r := []rune(s); len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}

// skipped elements hold nothing a reader would call the page's text.
var skipped = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Noscript: true, atom.Template: true, atom.Svg: true,
	atom.Iframe: true, atom.Head: true, atom.Nav: true, atom.Footer: true, atom.Form: true,
	atom.Button: true, atom.Select: true, atom.Canvas: true, atom.Object: true, atom.Embed: true,
	atom.Dialog: true, atom.Aside: true,
}

// block elements start a new line; inline ones run on.
var block = map[atom.Atom]bool{
	atom.P: true, atom.Div: true, atom.Br: true, atom.Li: true, atom.Tr: true, atom.Table: true,
	atom.H1: true, atom.H2: true, atom.H3: true, atom.H4: true, atom.H5: true, atom.H6: true,
	atom.Section: true, atom.Article: true, atom.Header: true, atom.Main: true, atom.Ul: true, atom.Ol: true,
	atom.Blockquote: true, atom.Pre: true, atom.Dt: true, atom.Dd: true, atom.Hr: true, atom.Figure: true,
	atom.Figcaption: true, atom.Details: true, atom.Summary: true, atom.Address: true,
}

// readable returns a page's title and its text: scripts, styles and page
// furniture removed, one line per block element. If leaving out navigation,
// footers and the like leaves next to nothing, it is all kept instead, since
// some pages put their whole content in such elements.
func readable(page []byte) (title, text string) {
	doc, err := html.Parse(bytes.NewReader(page))
	if err != nil {
		return "", ""
	}
	title = findTitle(doc)
	text = extract(doc, true)
	if len(text) < config.MinReadableChars {
		if all := extract(doc, false); len(all) > len(text) {
			text = all
		}
	}
	return title, text
}

func findTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.DataAtom == atom.Title {
		return clip(collapse(nodeText(n)), config.PageTitleChars)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if t := findTitle(c); t != "" {
			return t
		}
	}
	return ""
}

// neverText is skipped even when furniture is being kept.
var neverText = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Noscript: true, atom.Template: true, atom.Svg: true,
	atom.Head: true, atom.Iframe: true,
}

func extract(doc *html.Node, skipFurniture bool) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			b.WriteString(n.Data)
			return
		case html.ElementNode:
			if neverText[n.DataAtom] || (skipFurniture && skipped[n.DataAtom]) || hasAttr(n, "hidden") ||
				attr(n, "aria-hidden") == "true" {
				return
			}
			if block[n.DataAtom] {
				b.WriteByte('\n')
			}
		case html.CommentNode, html.DoctypeNode:
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode {
			switch {
			case block[n.DataAtom]:
				b.WriteByte('\n')
			case n.DataAtom == atom.Td || n.DataAtom == atom.Th:
				b.WriteByte('\t')
			}
		}
	}
	walk(doc)

	// Tidy: one space between words, no blank lines.
	var lines []string
	for _, line := range strings.Split(b.String(), "\n") {
		if line = collapse(line); line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}
