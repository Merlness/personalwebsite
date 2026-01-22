package main

import (
	"fmt"
	"os"
	"path/filepath"
	"personalwebsite/internal/images"
	"strings"
	"time"
)

func main() {
	start := time.Now()

	// Configuration
	// Assume running from project root
	contentRoot := "content/portfolio_optimized"
	if _, err := os.Stat(contentRoot); os.IsNotExist(err) {
		contentRoot = "content/portfolio"
	}

	cacheRoot := os.Getenv("CACHE_DIR")
	if cacheRoot == "" {
		cacheRoot = filepath.Join(os.TempDir(), "personalwebsite_cache")
	}

	if _, err := os.Stat(contentRoot); os.IsNotExist(err) {
		fmt.Printf("Error: %s not found. Please run from project root.\n", contentRoot)
		os.Exit(1)
	}

	fmt.Printf("Portfolio root: %s\n", contentRoot)
	fmt.Printf("Cache root: %s\n", cacheRoot)

	// Create resizer
	resizer := images.NewResizer(contentRoot, cacheRoot)

	// Find all images
	var imagesToProcess []string
	err := filepath.Walk(contentRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			// We need path relative to contentRoot
			relPath, err := filepath.Rel(contentRoot, path)
			if err != nil {
				return err
			}
			imagesToProcess = append(imagesToProcess, relPath)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error scanning files: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d images to process.\n", len(imagesToProcess))

	widths := []int{600, 1600}

	for _, relPath := range imagesToProcess {
		fmt.Printf("Processing %s... ", relPath)
		for _, w := range widths {
			_, err := resizer.Resize(relPath, w)
			if err != nil {
				fmt.Printf("\nError resizing %s to %d: %v\n", relPath, w, err)
			}
		}
		fmt.Println("Done")
	}

	fmt.Printf("Warmup complete in %v\n", time.Since(start))
}
