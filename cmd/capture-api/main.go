// capture-api is the Fly.io-hosted proxy the capture PWA writes through.
// See internal/captureapi for the handler.
package main

import (
	"cmp"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"personalwebsite/internal/captureapi"
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
	if cfg.AppToken == "" || cfg.GitHubToken == "" {
		log.Fatal("APP_TOKEN and GITHUB_TOKEN must be set")
	}

	addr := ":" + cmp.Or(os.Getenv("PORT"), "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           captureapi.NewHandler(cfg),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("capture-api listening on %s for %s/%s", addr, cfg.Owner, cfg.Repo)
	log.Fatal(srv.ListenAndServe())
}
