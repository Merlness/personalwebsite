package agent

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Runner is what the HTTP layer needs from the agent; *Agent implements it.
type Runner interface {
	Run(ctx context.Context, history []Turn, message string) (string, []Written, error)
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

// Unconfigured is served until the ANTHROPIC_API_KEY secret exists.
func Unconfigured() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"agent not configured: set the ANTHROPIC_API_KEY secret"}`, http.StatusServiceUnavailable)
	})
}
