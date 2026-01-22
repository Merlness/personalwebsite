package web

import (
	"net/http"
	"os"
	"path/filepath"
	"personalwebsite/internal/images"
	"strconv"
	"strings"
)

type ImageHandler struct {
	contentRoot string
	resizer     *images.Resizer
}

func NewImageHandler(contentRoot string) *ImageHandler {
	cacheRoot := os.Getenv("CACHE_DIR")
	if cacheRoot == "" {
		cacheRoot = filepath.Join(os.TempDir(), "personalwebsite_cache")
	}
	os.MkdirAll(cacheRoot, 0755)
	return &ImageHandler{
		contentRoot: contentRoot,
		resizer:     images.NewResizer(contentRoot, cacheRoot),
	}
}

func (h *ImageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	relPath := strings.TrimPrefix(r.URL.Path, "/")
	if relPath == "" {
		http.NotFound(w, r)
		return
	}

	fullPath := filepath.Join(h.contentRoot, relPath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	// Parse width query param
	widthStr := r.URL.Query().Get("w")
	if widthStr == "" {
		// Serve original file
		http.ServeFile(w, r, fullPath)
		return
	}

	width, err := strconv.Atoi(widthStr)
	if err != nil || width <= 0 || width > 4000 {
		// Invalid width, serve original
		http.ServeFile(w, r, fullPath)
		return
	}

	cachedPath, err := h.resizer.Resize(relPath, width)
	if err != nil {
		// Failed to resize, serve original as fallback
		http.ServeFile(w, r, fullPath)
		return
	}

	http.ServeFile(w, r, cachedPath)
}
