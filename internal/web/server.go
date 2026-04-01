package web

import (
	"net/http"
	"personalwebsite/internal/blog"
	"personalwebsite/internal/portfolio"
	"personalwebsite/internal/web/components"
	"strings"
)

type ServerConfig struct {
	PortfolioAssetsPath string
	AboutmeAssetsPath   string
	CSSAssetsPath       string
}

func NewServer(blogService blog.Service, portfolioService portfolio.Service, serverConfig ServerConfig) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/" {
			http.NotFound(writer, request)
			return
		}
		components.Home().Render(request.Context(), writer)
	})

	mux.HandleFunc("/about", func(writer http.ResponseWriter, request *http.Request) {
		components.About().Render(request.Context(), writer)
	})

	mux.HandleFunc("/portfolio", func(writer http.ResponseWriter, request *http.Request) {
		categories, err := portfolioService.GetCategories()
		if err != nil {
			http.Error(writer, "Failed to load portfolio categories", http.StatusInternalServerError)
			return
		}

		portfolioCats, adventureCats := portfolio.GroupCategories(categories)
		photoToBlog := map[string]string{}
		components.Portfolio(portfolioCats, adventureCats, photoToBlog).Render(request.Context(), writer)
	})

	mux.HandleFunc("GET /portfolio/{category}", func(writer http.ResponseWriter, request *http.Request) {
		categoryName := request.PathValue("category")
		category, err := portfolioService.GetCategory(categoryName)
		if err != nil {
			if err == portfolio.ErrCategoryNotFound {
				http.NotFound(writer, request)
				return
			}
			http.Error(writer, "Failed to load category", http.StatusInternalServerError)
			return
		}

		allCategories, err := portfolioService.GetCategories()
		if err != nil {
			http.Error(writer, "Failed to load categories", http.StatusInternalServerError)
			return
		}

		var photoToBlog map[string]string
		posts, err := blogService.GetAllPosts()
		if err == nil {
			photoToBlog = blog.BuildPhotoToBlogMap(posts)
		} else {
			photoToBlog = make(map[string]string)
		}

		components.PortfolioCategory(category, allCategories, photoToBlog).Render(request.Context(), writer)
	})

	mux.HandleFunc("GET /blog", func(writer http.ResponseWriter, request *http.Request) {
		posts, err := blogService.GetAllPosts()
		if err != nil {
			http.Error(writer, "Failed to load posts", http.StatusInternalServerError)
			return
		}
		components.BlogList(posts).Render(request.Context(), writer)
	})

	mux.HandleFunc("GET /blog/{slug}", func(writer http.ResponseWriter, request *http.Request) {
		slug := request.PathValue("slug")
		post, err := blogService.GetPost(slug)
		if err != nil {
			if err == blog.ErrPostNotFound {
				http.NotFound(writer, request)
				return
			}
			http.Error(writer, "Failed to load post", http.StatusInternalServerError)
			return
		}

		var prevPost, nextPost *blog.Post
		posts, postsErr := blogService.GetAllPosts()
		if postsErr == nil {
			prevPost, nextPost = blog.FindNeighbors(posts, slug)
		}

		components.BlogPost(post, prevPost, nextPost).Render(request.Context(), writer)
	})

	imageHandler := NewImageHandler(serverConfig.PortfolioAssetsPath)
	mux.Handle("/assets/portfolio/", http.StripPrefix("/assets/portfolio/", imageHandler))

	mux.Handle("/assets/aboutme/", http.StripPrefix("/assets/aboutme/", http.FileServer(http.Dir(serverConfig.AboutmeAssetsPath))))

	fileHandler := http.StripPrefix("/assets/", http.FileServer(http.Dir(serverConfig.CSSAssetsPath)))
	mux.Handle("/assets/", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasSuffix(request.URL.Path, ".css") {
			writer.Header().Set("Content-Type", "text/css")
		}
		fileHandler.ServeHTTP(writer, request)
	}))

	return mux
}
