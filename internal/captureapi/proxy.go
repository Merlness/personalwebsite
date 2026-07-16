// Package captureapi is a thin authenticated proxy in front of the GitHub
// Contents API, scoped to a single repository. The capture PWA talks to it
// instead of GitHub so the GitHub token lives server-side (a Fly secret)
// and the phone only holds an app token.
package captureapi

import (
	"crypto/subtle"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Config struct {
	// AppToken is the bearer token the PWA must present.
	AppToken string
	// GitHubToken authenticates the upstream call to api.github.com.
	GitHubToken string
	// Owner/Repo pin which repository may be touched. Requests for any
	// other repository are rejected.
	Owner string
	Repo  string
	// AllowedOrigins are echoed for CORS. Exact match per origin.
	AllowedOrigins []string
	// Upstream overrides the GitHub API base URL in tests.
	Upstream string
	// Client is the HTTP client for upstream calls; http.DefaultClient if nil.
	Client *http.Client
	// Agent, when set, is mounted at /agent behind the same app-token check.
	Agent http.Handler
}

func (c Config) upstream() string {
	if c.Upstream != "" {
		return strings.TrimSuffix(c.Upstream, "/")
	}
	return "https://api.github.com"
}

func (c Config) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return http.DefaultClient
}

func NewHandler(cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	mux.Handle("/repos/", proxyHandler(cfg))
	if cfg.Agent != nil {
		mux.Handle("/agent", requireAppToken(cfg, cfg.Agent))
	}
	return withCORS(cfg, mux)
}

func requireAppToken(cfg Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !authorized(cfg, r) {
			http.Error(w, `{"message":"bad app token"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withCORS(cfg Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		for _, allowed := range cfg.AllowedOrigins {
			if origin == allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, X-GitHub-Api-Version")
				w.Header().Set("Access-Control-Max-Age", "86400")
				break
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// Check before ServeMux, which would otherwise redirect ".." paths.
		if strings.Contains(r.URL.Path, "..") {
			http.Error(w, `{"message":"path not allowed"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func proxyHandler(cfg Config) http.Handler {
	prefix := fmt.Sprintf("/repos/%s/%s/contents/", cfg.Owner, cfg.Repo)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !authorized(cfg, r) {
			http.Error(w, `{"message":"bad app token"}`, http.StatusUnauthorized)
			return
		}
		switch r.Method {
		case http.MethodGet, http.MethodPut, http.MethodDelete:
		default:
			http.Error(w, `{"message":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		// Reject anything outside the pinned repo's contents endpoint,
		// including path traversal that would escape it after cleaning.
		if !strings.HasPrefix(r.URL.Path, prefix) || strings.Contains(r.URL.Path, "..") {
			http.Error(w, `{"message":"path not allowed"}`, http.StatusForbidden)
			return
		}

		up, err := http.NewRequestWithContext(r.Context(), r.Method,
			cfg.upstream()+r.URL.Path+"?"+r.URL.RawQuery, r.Body)
		if err != nil {
			http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
			return
		}
		up.Header.Set("Authorization", "Bearer "+cfg.GitHubToken)
		up.Header.Set("Accept", "application/vnd.github+json")
		up.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		res, err := cfg.client().Do(up)
		if err != nil {
			http.Error(w, `{"message":"upstream unreachable"}`, http.StatusBadGateway)
			return
		}
		defer res.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(res.StatusCode)
		io.Copy(w, res.Body)
	})
}

func authorized(cfg Config, r *http.Request) bool {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || cfg.AppToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(cfg.AppToken)) == 1
}
