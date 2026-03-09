package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"personalwebsite/internal/blog"
	"personalwebsite/internal/portfolio"
	"personalwebsite/internal/web/components"
)

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func renderPage(outputPath string, render func(context.Context, io.Writer) error) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating file %s: %w", outputPath, err)
	}
	defer file.Close()

	return render(context.Background(), file)
}

func generateHome(out string) error {
	return renderPage(filepath.Join(out, "index.html"), components.Home().Render)
}

func generateAbout(out string) error {
	return renderPage(filepath.Join(out, "about", "index.html"), components.About().Render)
}

func generatePortfolio(out string, pService portfolio.Service, bService blog.Service) error {
	categories, err := pService.GetCategories()
	if err != nil {
		return fmt.Errorf("loading portfolio categories: %w", err)
	}

	posts, err := bService.GetAllPosts()
	if err != nil {
		return fmt.Errorf("loading blog posts: %w", err)
	}

	photoToBlog := blog.BuildPhotoToBlogMap(posts)

	err = renderPage(filepath.Join(out, "portfolio", "index.html"), components.Portfolio(categories, photoToBlog).Render)
	if err != nil {
		return err
	}

	for _, cat := range categories {
		fullCat, err := pService.GetCategory(cat.Name)
		if err != nil {
			return fmt.Errorf("loading category %s: %w", cat.Name, err)
		}

		pagePath := filepath.Join(out, "portfolio", cat.Name, "index.html")
		err = renderPage(pagePath, components.PortfolioCategory(fullCat, categories, photoToBlog).Render)
		if err != nil {
			return err
		}
	}

	return nil
}

func generateBlog(out string, bService blog.Service) error {
	posts, err := bService.GetAllPosts()
	if err != nil {
		return fmt.Errorf("loading blog posts: %w", err)
	}

	err = renderPage(filepath.Join(out, "blog", "index.html"), components.BlogList(posts).Render)
	if err != nil {
		return err
	}

	for _, post := range posts {
		pagePath := filepath.Join(out, "blog", post.Slug, "index.html")
		err = renderPage(pagePath, components.BlogPost(post).Render)
		if err != nil {
			return err
		}
	}

	return nil
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

func main() {
	outputDir := "dist"

	fmt.Println("Building static site...")

	os.RemoveAll(outputDir)
	fatal(os.MkdirAll(outputDir, 0755))

	portfolioRoot := "content/portfolio_optimized"
	if _, err := os.Stat(portfolioRoot); os.IsNotExist(err) {
		log.Fatal("optimized portfolio not found. Run 'go run cmd/optimize/main.go' first.")
	}

	blogService := blog.NewFilesystemService("content/blog")
	portfolioService := portfolio.NewFilesystemService(portfolioRoot, "/assets/portfolio")

	fatal(generateHome(outputDir))
	fatal(generateAbout(outputDir))
	fatal(generatePortfolio(outputDir, portfolioService, blogService))
	fatal(generateBlog(outputDir, blogService))

	fatal(copyDir("internal/assets", filepath.Join(outputDir, "assets")))
	fatal(copyDir("content/portfolio_optimized", filepath.Join(outputDir, "assets/portfolio")))
	fatal(copyDir("content/aboutme_optimized", filepath.Join(outputDir, "assets/aboutme")))

	fatal(os.WriteFile(filepath.Join(outputDir, "CNAME"), []byte("merlmartin.com"), 0644))
	fatal(os.WriteFile(filepath.Join(outputDir, ".nojekyll"), []byte(""), 0644))

	fmt.Println("Build complete! The 'dist' folder is ready to deploy.")
}
