package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

func optimizeDir(sourceDir, destDir string, quality int) {
	fmt.Printf("Optimizing %s -> %s\n", sourceDir, destDir)

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		fmt.Printf("Warning: Source directory %s not found\n", sourceDir)
		return
	}

	// 1. Process new/modified files
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// check extension
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			return nil
		}

		// calculate relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Optimization targets
		// 1. Original (optimized) - used as fallback
		// 2. w600 - for grids
		// 3. w1600 - for lightbox
		targets := []struct {
			suffix string
			width  int
		}{
			{suffix: "", width: 2500},       // Base image, max 2500
			{suffix: "_w600", width: 600},   // Grid thumbnail
			{suffix: "_w1600", width: 1600}, // Lightbox view
		}

		for _, target := range targets {
			// Construct destination filename
			// e.g., image.jpg -> image.jpg (base)
			// e.g., image.jpg -> image_w600.jpg
			baseName := strings.TrimSuffix(relPath, filepath.Ext(relPath))
			destName := baseName + target.suffix + ext
			destPath := filepath.Join(destDir, destName)
			destDir := filepath.Dir(destPath)

			// Create dest dir
			if err := os.MkdirAll(destDir, 0755); err != nil {
				return err
			}

			// Check if needs update
			needsUpdate := true
			if destInfo, err := os.Stat(destPath); err == nil {
				if info.ModTime().Before(destInfo.ModTime()) {
					needsUpdate = false
				}
			}

			if !needsUpdate {
				continue
			}

			fmt.Printf("Processing %s (%s)... ", relPath, target.suffix)

			// Open image (only once ideally, but simple loop here)
			src, err := imaging.Open(path)
			if err != nil {
				fmt.Printf("Failed to open: %v\n", err)
				return nil
			}

			// Resize
			var dst *image.NRGBA
			if src.Bounds().Dx() > target.width {
				dst = imaging.Resize(src, target.width, 0, imaging.Lanczos)
			} else {
				dst = imaging.Clone(src)
			}

			// Save
			file, err := os.Create(destPath)
			if err != nil {
				fmt.Printf("Failed to create file: %v\n", err)
				continue
			}

			if ext == ".png" {
				err = png.Encode(file, dst)
			} else {
				err = jpeg.Encode(file, dst, &jpeg.Options{Quality: quality})
			}
			file.Close()

			if err != nil {
				fmt.Printf("Failed to save: %v\n", err)
			} else {
				fmt.Println("Done")
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking source dir: %v\n", err)
	}

	// 2. Cleanup (Simplified for now - just manual for safety in this change)
}

func main() {
	jpegQuality := 85

	optimizeDir("content/portfolio", "content/portfolio_optimized", jpegQuality)
	optimizeDir("content/aboutme", "content/aboutme_optimized", jpegQuality)

	fmt.Println("Optimization complete.")
}
