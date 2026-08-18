package presentation

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// sseWriter frames Server-Sent Events for AIHandler.Chat — one small,
// module-local helper rather than a shared platform package, since nothing
// else in this codebase streams yet (see the Phase 2 RFC).
type sseWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// newSSEWriter switches w into SSE framing: sets headers and flushes them
// immediately so the client's connection opens right away. Returns ok=false
// if w doesn't support flushing (shouldn't happen with chi's default
// transport, but every real external-facing write path should still check
// rather than assume).
func newSSEWriter(w http.ResponseWriter) (*sseWriter, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, false
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Disable reverse-proxy response buffering (nginx honors this header) —
	// without it, deltas can sit in a proxy buffer instead of reaching the
	// client as they're written, defeating the point of streaming.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	return &sseWriter{w: w, flusher: flusher}, true
}

func (s *sseWriter) send(event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		data = []byte(`{}`)
	}
	fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, data)
	s.flusher.Flush()
}

func (s *sseWriter) sendDelta(content string) {
	s.send("delta", map[string]string{"content": content})
}

func (s *sseWriter) sendDone(blocked bool) {
	s.send("done", map[string]bool{"blocked": blocked})
}

// sendError sends a generic, client-safe message — never err.Error() itself,
// same "don't leak internal detail" rule apperr.NewInternalError follows on
// the non-streaming error path.
func (s *sseWriter) sendError() {
	s.send("error", map[string]string{"message": "AI tạm thời không khả dụng, vui lòng thử lại sau."})
}
