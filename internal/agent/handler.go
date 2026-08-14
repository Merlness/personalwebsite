package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// Runner is what the HTTP layer needs from the agent; *Agent implements it.
type Runner interface {
	Run(ctx context.Context, history []Turn, message string) (string, []Written, error)
}

// StreamRunner is the optional half: a Runner that can report progress while
// the run is still going. *Agent implements it, and the handler falls back to
// a plain JSON reply for any Runner that does not.
type StreamRunner interface {
	Stream(ctx context.Context, history []Turn, message string, emit func(Event)) (string, []Written, error)
}

// Handler serves POST /agent. Authentication is the caller's job (the
// captureapi mux wraps it with the app-token check).
type Handler struct {
	Runner  Runner
	Timeout time.Duration
}

type request struct {
	Message string `json:"message"`
	History []Turn `json:"history"`
}

type response struct {
	Reply   string    `json:"reply"`
	Written []Written `json:"written"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"message":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var req request
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, `{"message":"bad request body"}`, http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, `{"message":"message is required"}`, http.StatusBadRequest)
		return
	}
	if len(req.History) > 20 {
		req.History = req.History[len(req.History)-20:]
	}

	timeout := h.Timeout
	if timeout == 0 {
		timeout = 55 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	if sr, flusher, ok := h.streaming(w, r); ok {
		h.serveStream(ctx, w, flusher, sr, req)
		return
	}

	reply, written, err := h.Runner.Run(ctx, req.History, req.Message)
	if err != nil {
		log.Printf("agent run failed: %v", err)
		http.Error(w, `{"message":"agent failed, try again"}`, http.StatusBadGateway)
		return
	}
	if written == nil {
		written = []Written{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response{Reply: reply, Written: written})
}

// streaming reports whether this request can and should be served as SSE. All
// three have to hold: the client asked for it, the runner supports it, and the
// response writer can flush.
func (h *Handler) streaming(w http.ResponseWriter, r *http.Request) (StreamRunner, http.Flusher, bool) {
	if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		return nil, nil, false
	}
	sr, ok := h.Runner.(StreamRunner)
	if !ok {
		return nil, nil, false
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, nil, false
	}
	return sr, flusher, true
}

// serveStream writes the run as Server-Sent Events. Every progress event is one
// "data:" frame; the final frame is named "done" and carries the same JSON body
// the non-streaming path returns, so the client has one shape to store.
func (h *Handler) serveStream(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, sr StreamRunner, req request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Tell any proxy in front of us not to buffer, which would defeat the point.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Stream calls emit synchronously from this goroutine, so no lock is needed.
	// json.Marshal escapes newlines, so a frame is always a single line.
	frame := func(name string, v any) {
		b, err := json.Marshal(v)
		if err != nil {
			return
		}
		if name != "" {
			fmt.Fprintf(w, "event: %s\n", name)
		}
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	reply, written, err := sr.Stream(ctx, req.History, req.Message, func(e Event) { frame("", e) })
	if err != nil {
		log.Printf("agent stream failed: %v", err)
		frame("", Event{Type: EventError, Text: "agent failed, try again"})
		return
	}
	if written == nil {
		written = []Written{}
	}
	frame("done", response{Reply: reply, Written: written})
}

// Unconfigured is served until the ANTHROPIC_API_KEY secret exists.
func Unconfigured() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"agent not configured: set the ANTHROPIC_API_KEY secret"}`, http.StatusServiceUnavailable)
	})
}
