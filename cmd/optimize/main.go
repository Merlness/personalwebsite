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

func optimizeDir(sourceDir, destDir string, maxWidth, quality int) {
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

		destPath := filepath.Join(destDir, relPath)
		destDir := filepath.Dir(destPath)

		// Create dest dir
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}

		// Check if needs update (missing or older)
		needsUpdate := true
		if destInfo, err := os.Stat(destPath); err == nil {
			// If dest exists, check if source is newer
			if info.ModTime().Before(destInfo.ModTime()) {
				needsUpdate = false
			}
		}

		if !needsUpdate {
			return nil
		}

		fmt.Printf("Processing %s... ", relPath)

		// Open image
		src, err := imaging.Open(path)
		if err != nil {
			fmt.Printf("Failed to open: %v\n", err)
			return nil
		}

		// Resize if larger than maxWidth
		var dst *image.NRGBA
		if src.Bounds().Dx() > maxWidth {
			dst = imaging.Resize(src, maxWidth, 0, imaging.Lanczos)
		} else {
			dst = imaging.Clone(src)
		}

		// Save
		file, err := os.Create(destPath)
		if err != nil {
			fmt.Printf("Failed to create file: %v\n", err)
			return nil
		}
		defer file.Close()

		if ext == ".png" {
			err = png.Encode(file, dst)
		} else {
			err = jpeg.Encode(file, dst, &jpeg.Options{Quality: quality})
		}

		if err != nil {
			fmt.Printf("Failed to save: %v\n", err)
		} else {
			fmt.Println("Done")
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking source dir: %v\n", err)
	}

	// 2. Cleanup deleted files
	fmt.Println("Cleaning up deleted files...")
	err = filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // ignore errors here
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(destDir, path)
		if err != nil {
			return nil
		}

		sourcePath := filepath.Join(sourceDir, relPath)
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			fmt.Printf("Removing orphan file: %s\n", relPath)
			os.Remove(path)
		}
		return nil
	})
}

func main() {
	// Max width for "high quality" web display
	maxWidth := 2500
	jpegQuality := 85

	optimizeDir("content/portfolio", "content/portfolio_optimized", maxWidth, jpegQuality)
	optimizeDir("content/aboutme", "content/aboutme_optimized", maxWidth, jpegQuality)

	fmt.Println("Optimization complete. You can now commit the optimized directories.")
}
