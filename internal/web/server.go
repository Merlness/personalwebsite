package web

import (
	"net/http"
	"os"
	"personalwebsite/internal/blog"
	"personalwebsite/internal/portfolio"
	"personalwebsite/internal/web/components"
	"strings"
)

func NewServer(blogService blog.Service, portfolioService portfolio.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Basic exact matching for root to avoid catch-all behavior for unhandled paths
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		component := components.Home()
		component.Render(r.Context(), w)
	})

	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		component := components.About()
		component.Render(r.Context(), w)
	})

	mux.HandleFunc("/portfolio", func(w http.ResponseWriter, r *http.Request) {
		categories, err := portfolioService.GetCategories()
		if err != nil {
			http.Error(w, "Failed to load portfolio categories", http.StatusInternalServerError)
			return
		}

		// photoToBlog is unused in the main portfolio view now
		photoToBlog := map[string]string{}

		component := components.Portfolio(categories, photoToBlog)
		component.Render(r.Context(), w)
	})

	mux.HandleFunc("GET /portfolio/{category}", func(w http.ResponseWriter, r *http.Request) {
		categoryName := r.PathValue("category")
		category, err := portfolioService.GetCategory(categoryName)
		if err != nil {
			if err == portfolio.ErrCategoryNotFound {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "Failed to load category", http.StatusInternalServerError)
			return
		}

		// fetch all categories for "More Collections"
		allCategories, err := portfolioService.GetCategories()
		if err != nil {
			http.Error(w, "Failed to load categories", http.StatusInternalServerError)
			return
		}

		// Build photoToBlog map (reuse logic)
		photoToBlog := make(map[string]string)
		posts, err := blogService.GetAllPosts()
		if err == nil {
			for _, post := range posts {
				for _, photo := range post.LinkedPhotos {
					photoToBlog[photo] = post.Slug
				}
			}
		}

		component := components.PortfolioCategory(category, allCategories, photoToBlog)
		component.Render(r.Context(), w)
	})

	mux.HandleFunc("GET /blog", func(w http.ResponseWriter, r *http.Request) {
		posts, err := blogService.GetAllPosts()
		if err != nil {
			http.Error(w, "Failed to load posts", http.StatusInternalServerError)
			return
		}
		component := components.BlogList(posts)
		component.Render(r.Context(), w)
	})

	mux.HandleFunc("GET /blog/{slug}", func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		post, err := blogService.GetPost(slug)
		if err != nil {
			if err == blog.ErrPostNotFound {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "Failed to load post", http.StatusInternalServerError)
			return
		}
		component := components.BlogPost(post)
		component.Render(r.Context(), w)
	})

	portfolioPath := "./content/portfolio"
	if _, err := os.Stat(portfolioPath); os.IsNotExist(err) {
		portfolioPath = "../../content/portfolio"
	}

	// Use ImageHandler instead of standard FileServer
	imageHandler := NewImageHandler(portfolioPath)
	mux.Handle("/assets/portfolio/", http.StripPrefix("/assets/portfolio/", imageHandler))

	mux.Handle("/assets/aboutme/", http.StripPrefix("/assets/aboutme/", http.FileServer(http.Dir("content/aboutme"))))

	assetsPath := "./internal/assets"
	if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
		assetsPath = "../assets"
	}

	fileHandler := http.StripPrefix("/assets/", http.FileServer(http.Dir(assetsPath)))
	mux.Handle("/assets/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		}
		fileHandler.ServeHTTP(w, r)
	}))

	return mux
}
