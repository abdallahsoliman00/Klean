package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/ssestream"

	"klean/backend/internal/config"
	"klean/backend/internal/prompt"
)

// UpstreamError carries a message meant for the person using the UI.
type UpstreamError struct {
	Msg string
	// rejected is set when the provider refused the request itself (a 4xx or
	// 501), as opposed to failing while answering it.
	rejected bool
	// missing is set when the provider has no such endpoint at all.
	missing bool
}

func (e *UpstreamError) Error() string { return e.Msg }

func upstream(format string, a ...any) error { return &UpstreamError{Msg: fmt.Sprintf(format, a...)} }

// endpointMissing reports whether err says the provider has no such endpoint.
func endpointMissing(err error) bool {
	var ue *UpstreamError
	return errors.As(err, &ue) && ue.missing
}

// noSuchEndpoint tells a route the server doesn't have (404 or 405) from a
// model it doesn't have, which is also a 404 but names the model.
func noSuchEndpoint(status int, detail string) bool {
	if status == http.StatusMethodNotAllowed {
		return true
	}
	return status == http.StatusNotFound && !strings.Contains(strings.ToLower(detail), "model")
}

// How providers word the refusal of something a request carried. They differ
// ("tools is not supported", "does not support function calling", "unknown
// field tools"; "image input is not supported", "not a multimodal model",
// "content must be a string"), so any 4xx or 501 with one of the words counts.
var (
	toolWords  = []string{"tool", "function"}
	imageWords = []string{"image", "vision", "multimodal", "multi-modal", "must be a string"}
)

const (
	toolsAdvice = "This model or provider doesn't appear to support tools. Turn tools off in the composer " +
		"and send the message again."
	imagesAdvice = "This model or server can't read images. Pick a model with vision support, " +
		"or send the message without the images."
)

// explainRequest rewrites a provider's rejection of the images or the tools
// that r carried into advice. Images are checked first: a model without
// vision is the likelier cause when a request has both.
func explainRequest(err error, r Request) error {
	if hasImages(r.Turns) {
		if advised := advise(err, imageWords, imagesAdvice); advised != err {
			return advised
		}
	}
	if len(r.Tools) > 0 {
		return advise(err, toolWords, toolsAdvice)
	}
	return err
}

func hasImages(turns []prompt.Turn) bool {
	for _, t := range turns {
		if len(t.Images) > 0 {
			return true
		}
	}
	return false
}

// advise returns the advice, with the provider's own words after it, when err
// is a rejected request that mentions one of the words; otherwise err.
func advise(err error, words []string, advice string) error {
	var ue *UpstreamError
	if !errors.As(err, &ue) || !ue.rejected {
		return err
	}
	low := strings.ToLower(ue.Msg)
	for _, w := range words {
		if strings.Contains(low, w) {
			return upstream("%s (%s)", advice, strings.TrimSuffix(ue.Msg, "."))
		}
	}
	return err
}

// explain turns an SDK or network error into a message for the user.
func explain(err error, url string) error {
	if errors.Is(err, context.Canceled) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return upstream("The provider stopped responding (timed out).")
	}
	var se *ssestream.StreamError
	if errors.As(err, &se) {
		return upstream("The provider reported an error: %s", errorDetail([]byte(se.Event.Data)))
	}
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		detail := apiErr.Message
		if detail == "" && apiErr.Response != nil && apiErr.Response.Body != nil {
			body, _ := io.ReadAll(io.LimitReader(apiErr.Response.Body, config.ErrorBodyBytes))
			detail = errorDetail(body)
		}
		switch {
		case apiErr.StatusCode == http.StatusUnauthorized:
			return upstream("The provider rejected the API key (401). Check the key in Providers & settings.")
		case noSuchEndpoint(apiErr.StatusCode, detail):
			return &UpstreamError{missing: true, Msg: fmt.Sprintf(
				"%s returned %d. Check that the base URL ends where the API starts (often /v1).", url, apiErr.StatusCode)}
		}
		msg := fmt.Sprintf("The provider returned %d.", apiErr.StatusCode)
		if detail != "" {
			msg = fmt.Sprintf("The provider returned %d: %s", apiErr.StatusCode, detail)
		}
		refused := apiErr.StatusCode == http.StatusNotImplemented ||
			(apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 && apiErr.StatusCode != http.StatusTooManyRequests)
		return &UpstreamError{Msg: msg, rejected: refused}
	}
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return upstream("%s didn't return JSON. Is this an OpenAI-compatible endpoint?", url)
	}
	return upstream("Couldn't reach %s. Check that the server is running and the URL is right.", url)
}

// errorDetail pulls a readable message out of an error body.
func errorDetail(body []byte) string {
	var v map[string]any
	if json.Unmarshal(body, &v) == nil {
		for _, key := range []string{"error", "detail", "message"} {
			switch e := v[key].(type) {
			case string:
				return e
			case map[string]any:
				if msg, ok := e["message"].(string); ok {
					return msg
				}
			}
		}
	}
	s := strings.TrimSpace(string(body))
	if r := []rune(s); len(r) > config.ErrorDetailChars {
		s = string(r[:config.ErrorDetailChars])
	}
	return s
}
