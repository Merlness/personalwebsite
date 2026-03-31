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

// allowedWidths restricts resize operations to the specific widths we actually use.
// This prevents DoS via arbitrary resize requests on large images.
var allowedWidths = map[int]bool{600: true, 1200: true, 1600: true}

func (h *ImageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	relPath := strings.TrimPrefix(r.URL.Path, "/")
	if relPath == "" {
		http.NotFound(w, r)
		return
	}

	// Security: prevent directory traversal attacks
	if strings.Contains(relPath, "..") {
		http.NotFound(w, r)
		return
	}

	fullPath := filepath.Join(h.contentRoot, relPath)

	// Security: verify the resolved path stays within content root
	absRoot, err := filepath.Abs(h.contentRoot)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if !strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) {
		http.NotFound(w, r)
		return
	}

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
	if err != nil || !allowedWidths[width] {
		// Invalid or non-whitelisted width, serve original
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
