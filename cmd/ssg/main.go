package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"personalwebsite/internal/blog"
	"personalwebsite/internal/portfolio"
	"personalwebsite/internal/web/components"
)

func main() {
	outputDir := "dist"

	fmt.Println("Building static site...")

	// Cleanup
	os.RemoveAll(outputDir)
	os.MkdirAll(outputDir, 0755)

	// Setup Services
	// Note: We point to the OPTIMIZED directory now
	portfolioRoot := "content/portfolio_optimized"
	if _, err := os.Stat(portfolioRoot); os.IsNotExist(err) {
		fmt.Println("Error: optimized portfolio not found. Run 'go run cmd/optimize/main.go' first.")
		os.Exit(1)
	}

	blogService := blog.NewFilesystemService("content/blog")
	// Web path prefix must be relative from root of site
	portfolioService := portfolio.NewFilesystemService(portfolioRoot, "/assets/portfolio")

	// 1. Generate Pages
	generateHome(outputDir)
	generateAbout(outputDir)
	generatePortfolio(outputDir, portfolioService, blogService)
	generateBlog(outputDir, blogService)

	// 2. Copy Assets
	copyDir("internal/assets", filepath.Join(outputDir, "assets"))
	copyDir("content/portfolio_optimized", filepath.Join(outputDir, "assets/portfolio"))
	copyDir("content/aboutme_optimized", filepath.Join(outputDir, "assets/aboutme"))

	// 3. Create CNAME if needed (optional)
	os.WriteFile(filepath.Join(outputDir, "CNAME"), []byte("merlmartin.com"), 0644)

	// 4. Create .nojekyll to prevent GitHub from ignoring underscore files
	os.WriteFile(filepath.Join(outputDir, ".nojekyll"), []byte(""), 0644)

	fmt.Println("Build complete! The 'dist' folder is ready to deploy.")
}

func generateHome(out string) {
	f, _ := os.Create(filepath.Join(out, "index.html"))
	defer f.Close()
	components.Home().Render(context.Background(), f)
}

func generateAbout(out string) {
	// Create directory if needed for pretty URLs (about/index.html)
	os.MkdirAll(filepath.Join(out, "about"), 0755)
	f, _ := os.Create(filepath.Join(out, "about", "index.html"))
	defer f.Close()
	components.About().Render(context.Background(), f)
}

func generatePortfolio(out string, pService portfolio.Service, bService blog.Service) {
	os.MkdirAll(filepath.Join(out, "portfolio"), 0755)

	// Main Portfolio Page
	categories, _ := pService.GetCategories()
	// Build photoToBlog map (for reading stories from photos)
	photoToBlog := make(map[string]string)
	posts, _ := bService.GetAllPosts()
	for _, post := range posts {
		for _, photo := range post.LinkedPhotos {
			photoToBlog[photo] = post.Slug
		}
	}

	f, _ := os.Create(filepath.Join(out, "portfolio", "index.html"))
	components.Portfolio(categories, photoToBlog).Render(context.Background(), f)
	f.Close()

	// Category Pages
	for _, cat := range categories {
		catDir := filepath.Join(out, "portfolio", cat.Name)
		os.MkdirAll(catDir, 0755)

		f, _ := os.Create(filepath.Join(catDir, "index.html"))
		// Re-fetch detailed category to ensure we have images
		fullCat, _ := pService.GetCategory(cat.Name)

		components.PortfolioCategory(fullCat, categories, photoToBlog).Render(context.Background(), f)
		f.Close()
	}
}

func generateBlog(out string, bService blog.Service) {
	os.MkdirAll(filepath.Join(out, "blog"), 0755)

	posts, _ := bService.GetAllPosts()

	// Blog Index
	f, _ := os.Create(filepath.Join(out, "blog", "index.html"))
	components.BlogList(posts).Render(context.Background(), f)
	f.Close()

	// Individual Posts
	for _, post := range posts {
		postDir := filepath.Join(out, "blog", post.Slug)
		os.MkdirAll(postDir, 0755)

		f, _ := os.Create(filepath.Join(postDir, "index.html"))
		components.BlogPost(post).Render(context.Background(), f)
		f.Close()
	}
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Copy file
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}
