package server

import (
	"context"
	"log"
	"strings"

	"klean/backend/internal/config"
	"klean/backend/internal/llm"
	"klean/backend/internal/prompt"
)

// summaryTitle asks the chat's model to name the chat after its first
// message and reply, and saves the answer over the provisional title. It
// returns "" when the provisional title stays: the request failed or took too
// long, the model wrote nothing usable, or the user renamed the chat meanwhile.
func (s *Server) summaryTitle(ctx context.Context, chatID string, prep *prepared, reply string) string {
	ctx, cancel := context.WithTimeout(ctx, config.TitleTimeout)
	defer cancel()

	req := prep.req
	req.Instructions, req.Tools = config.TitleInstructions, nil
	req.Turns = prompt.TitleTurns(prep.userMsg.Content, reply)
	var raw strings.Builder
	if _, err := llm.StreamReply(ctx, req, llm.Handler{OnDelta: func(text string) { raw.WriteString(text) }}); err != nil {
		log.Printf("naming chat: %v", err)
		return ""
	}
	title := prompt.CleanTitle(raw.String())
	if title == "" {
		return ""
	}
	changed, err := s.db.ReplaceTitle(chatID, *prep.title, title)
	if err != nil {
		log.Printf("saving chat title: %v", err)
	}
	if !changed {
		return ""
	}
	return title
}
