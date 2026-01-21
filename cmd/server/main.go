package main

import (
	"fmt"
	"net/http"
	"os"
	"personalwebsite/internal/blog"
	"personalwebsite/internal/portfolio"
	"personalwebsite/internal/web"
)

func main() {
	blogService := blog.NewFilesystemService("content/blog")
	portfolioService := portfolio.NewFilesystemService("content/portfolio", "/assets/portfolio")
	server := web.NewServer(blogService, portfolioService)
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
