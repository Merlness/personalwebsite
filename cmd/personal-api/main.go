// personal-api is the Fly.io service behind Merl's organizer PWA: the
// authenticated GitHub proxy for file reads and writes, plus the /agent
// endpoint that runs real-time organizer commands through the Anthropic API.
package main

import (
	"cmp"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"

	"personalwebsite/internal/agent"
	"personalwebsite/internal/auth"
	"personalwebsite/internal/captureapi"
	"personalwebsite/internal/lifeorg"
)

func main() {
	cfg := captureapi.Config{
		AppToken:    os.Getenv("APP_TOKEN"),
		GitHubToken: os.Getenv("GITHUB_TOKEN"),
		Owner:       cmp.Or(os.Getenv("REPO_OWNER"), "Merlness"),
		Repo:        cmp.Or(os.Getenv("REPO_NAME"), "life-organizer"),
		AllowedOrigins: strings.Split(
			cmp.Or(os.Getenv("ALLOWED_ORIGINS"), "https://merlmartin.com"), ","),
	}

	var handler http.Handler
	if cfg.AppToken == "" || cfg.GitHubToken == "" {
		// Boot healthy so a fresh deploy does not crash-loop; secrets can be
		// set afterwards (fly secrets set restarts the machine with them).
		log.Print("WARNING: APP_TOKEN and/or GITHUB_TOKEN missing; serving 503s until secrets are set")
		handler = degraded()
	} else {
		cfg.Agent = buildAgent(cfg)
		if a := buildAuth(); a != nil {
			cfg.Auth = a.Handler()
			cfg.Session = a.Current
		}
		handler = captureapi.NewHandler(cfg)
	}

	addr := ":" + cmp.Or(os.Getenv("PORT"), "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       20 * time.Second,
		IdleTimeout:       120 * time.Second,
		// No WriteTimeout: an /agent reply can legitimately run up to the
		// handler's 55s budget, and a short WriteTimeout would truncate it.
	}
	log.Printf("personal-api listening on %s for %s/%s", addr, cfg.Owner, cfg.Repo)
	log.Fatal(srv.ListenAndServe())
}

func buildAgent(cfg captureapi.Config) http.Handler {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		log.Print("WARNING: ANTHROPIC_API_KEY missing; /agent serves 503 until it is set")
		return agent.Unconfigured()
	}
	store := &lifeorg.Client{
		Token:  cfg.GitHubToken,
		Owner:  cfg.Owner,
		Repo:   cfg.Repo,
		Branch: cmp.Or(os.Getenv("REPO_BRANCH"), "main"),
	}
	a := &agent.Agent{
		Client: anthropic.NewClient(), // reads ANTHROPIC_API_KEY from the environment
		Store:  store,
		Model:  cmp.Or(os.Getenv("AGENT_MODEL"), "claude-haiku-4-5"),
	}
	return &agent.Handler{Runner: a}
}

// buildAuth returns the configured Google sign-in, or nil when the OAuth
// client and session key are unset (the app-token path stays in force).
func buildAuth() *auth.Auth {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	sessionKey := os.Getenv("SESSION_KEY")
	if clientID == "" || clientSecret == "" || sessionKey == "" {
		log.Print("auth: GOOGLE_CLIENT_ID/SECRET or SESSION_KEY unset; Google sign-in disabled")
		return nil
	}
	a, err := auth.New(auth.Config{
		ClientID:      clientID,
		ClientSecret:  clientSecret,
		SessionKey:    []byte(sessionKey),
		RedirectURL:   cmp.Or(os.Getenv("OAUTH_REDIRECT_URL"), "https://api.merlmartin.com/auth/callback"),
		SuccessURL:    cmp.Or(os.Getenv("OAUTH_SUCCESS_URL"), "https://merlmartin.com/capture/"),
		CookieDomain:  cmp.Or(os.Getenv("COOKIE_DOMAIN"), "merlmartin.com"),
		AllowedEmails: strings.Split(cmp.Or(os.Getenv("ALLOWED_EMAILS"), "mmartin777@gmail.com,merl@bennusystems.com"), ","),
	})
	if err != nil {
		log.Printf("WARNING: auth disabled: %v", err)
		return nil
	}
	log.Print("auth: Google sign-in enabled")
	return a
}

func degraded() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"server not configured: set GITHUB_TOKEN, APP_TOKEN, and ANTHROPIC_API_KEY secrets"}`, http.StatusServiceUnavailable)
	})
	return mux
}
