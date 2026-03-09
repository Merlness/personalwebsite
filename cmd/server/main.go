package main

import (
	"fmt"
	"net/http"
	"os"
	"personalwebsite/internal/blog"
	"personalwebsite/internal/config"
	"personalwebsite/internal/portfolio"
	"personalwebsite/internal/web"
)

func main() {
	blogService := blog.NewFilesystemService("content/blog")
	portfolioService := portfolio.NewFilesystemService(config.ResolvePortfolioRoot(), "/assets/portfolio")

	serverConfig := web.ServerConfig{
		PortfolioAssetsPath: config.ResolvePortfolioRoot(),
		AboutmeAssetsPath:   config.ResolveAboutmeRoot(),
		CSSAssetsPath:       "internal/assets",
	}

	server := web.NewServer(blogService, portfolioService, serverConfig)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s\n", port)
	if err := http.ListenAndServe(":"+port, server); err != nil {
		fmt.Fprintf(os.Stderr, "Server failed to start: %v\n", err)
		os.Exit(1)
	}
}
