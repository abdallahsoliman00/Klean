package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// sseEvent is one Server-Sent Event, its data already encoded.
type sseEvent struct {
	name string
	data []byte
}

// replayOmits are the events a client attaching to a running reply is not
// sent again: their messages are saved before the reply starts, so the chat
// it loaded already has them.
var replayOmits = map[string]bool{"user": true, "answered": true}

// run is one reply being written. It belongs to the server, not to the
// request that started it, so it carries on when that client goes away. Its
// events are kept in order for anyone who attaches later; the log is bounded
// by the one reply it holds and is dropped with the run.
type run struct {
	ctx    context.Context
	cancel context.CancelFunc
	// saved is closed once the reply is in the database (or there turned out
	// to be nothing to write): from then on the chat is not running.
	saved chan struct{}

	mu     sync.Mutex
	events []sseEvent
	ended  bool          // no more events will come
	more   chan struct{} // closed, and replaced, whenever events or ended change
}

// send records an event. It never waits for a reader.
func (rn *run) send(event string, data any) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(data)
	rn.mu.Lock()
	defer rn.mu.Unlock()
	rn.events = append(rn.events, sseEvent{event, bytes.TrimSpace(buf.Bytes())})
	rn.wake()
}

// wake tells the readers there is something new. The caller holds rn.mu.
func (rn *run) wake() {
	close(rn.more)
	rn.more = make(chan struct{})
}

// from returns the events after the first n, whether those are the last, and
// a channel that is closed when that answer changes.
func (rn *run) from(n int) ([]sseEvent, bool, <-chan struct{}) {
	rn.mu.Lock()
	defer rn.mu.Unlock()
	return rn.events[n:], rn.ended, rn.more
}

// streamTo writes the run's events to the client, from the first one, until
// the run ends or the client goes away. A slow client only holds itself up.
func (rn *run) streamTo(w http.ResponseWriter, r *http.Request, omit map[string]bool) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	for sent := 0; ; {
		events, ended, more := rn.from(sent)
		for _, e := range events {
			if !omit[e.name] {
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", e.name, e.data)
			}
		}
		sent += len(events)
		_ = rc.Flush()
		if ended {
			return
		}
		select {
		case <-more:
		case <-r.Context().Done():
			return
		}
	}
}

// startRun claims the chat for a new reply. A chat has one reply at a time.
func (s *Server) startRun(chatID string) (*run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case s.closed:
		return nil, errorf(http.StatusServiceUnavailable, "Klean is shutting down.")
	case s.runs[chatID] != nil:
		return nil, errorf(http.StatusConflict, "This chat is still writing a reply. Stop it or wait for it to finish first.")
	}
	rn := &run{saved: make(chan struct{}), more: make(chan struct{})}
	rn.ctx, rn.cancel = context.WithCancel(s.ctx)
	s.runs[chatID] = rn
	s.active.Add(1)
	return rn, nil
}

// runOf returns the chat's running reply, or nil.
func (s *Server) runOf(chatID string) *run {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.runs[chatID]
}

// markSaved ends the chat's running state: its reply is in the database.
func (s *Server) markSaved(chatID string, rn *run) {
	s.mu.Lock()
	delete(s.runs, chatID)
	s.mu.Unlock()
	close(rn.saved)
}

// endRun closes the run's stream once its last event has been sent.
func (s *Server) endRun(rn *run) {
	rn.cancel()
	rn.mu.Lock()
	rn.ended = true
	rn.wake()
	rn.mu.Unlock()
	s.active.Done()
}

// stopRun stops the chat's reply, if one is running, and returns it.
func (s *Server) stopRun(chatID string) *run {
	rn := s.runOf(chatID)
	if rn != nil {
		rn.cancel()
	}
	return rn
}

// Close stops the replies being written and returns once what they had so
// far is saved. Call it before closing the database.
func (s *Server) Close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	s.cancel()
	s.active.Wait()
}

// streamReply attaches to the chat's running reply: everything it has sent so
// far, then the rest as it comes. 204 when no reply is running.
func (s *Server) streamReply(w http.ResponseWriter, r *http.Request) {
	rn := s.runOf(r.PathValue("id"))
	if rn == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	rn.streamTo(w, r, replayOmits)
}

// stopReply stops the chat's running reply and answers once the part of it
// that arrived is saved, so the chat can be loaded right after. 204 also when
// nothing was running.
func (s *Server) stopReply(w http.ResponseWriter, r *http.Request) {
	if rn := s.stopRun(r.PathValue("id")); rn != nil {
		select {
		case <-rn.saved:
		case <-r.Context().Done():
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
