package server

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"klean/backend/internal/config"
	"klean/backend/internal/store"
)

// parseFiles checks a message's text files: their number, size and that they
// are text. Names are cut to the base name, as the model only needs that.
func parseFiles(in []store.File) ([]store.File, error) {
	if len(in) > config.MaxFilesPerMessage {
		return nil, errorf(http.StatusUnprocessableEntity, "A message can have at most %d files.", config.MaxFilesPerMessage)
	}
	out := make([]store.File, 0, len(in))
	for i, f := range in {
		name := cleanTitle(filepath.Base(strings.ReplaceAll(f.Name, `\`, "/")))
		if r := []rune(name); len(r) > config.FileNameMaxChars {
			name = string(r[:config.FileNameMaxChars])
		}
		if name == "" || name == "." || name == "/" {
			name = fmt.Sprintf("file-%d.txt", i+1)
		}
		if len(f.Text) > config.MaxFileBytes {
			return nil, errorf(http.StatusUnprocessableEntity, "%s is too large: a file may be at most %d KB.", name, config.MaxFileBytes>>10)
		}
		if !utf8.ValidString(f.Text) || strings.ContainsRune(f.Text, 0) {
			return nil, errorf(http.StatusUnprocessableEntity, "%s isn't a text file. Only text files can be attached.", name)
		}
		out = append(out, store.File{Name: name, Text: f.Text})
	}
	return out, nil
}

// withFiles is a user message as the model reads it: each attached file in a
// <file> block, then the message's own text.
func withFiles(content string, files []store.File) string {
	if len(files) == 0 {
		return content
	}
	var b strings.Builder
	for _, f := range files {
		fmt.Fprintf(&b, "<file name=%q>\n%s\n</file>\n\n", f.Name, strings.TrimRight(f.Text, "\n"))
	}
	b.WriteString(content)
	return b.String()
}
