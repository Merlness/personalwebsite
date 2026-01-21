package images

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestResizer_Resize(t *testing.T) {
	// Setup temp dirs
	contentRoot := t.TempDir()
	cacheRoot := t.TempDir()

	// Create a dummy image
	relPath := "test_image.png"
	fullPath := filepath.Join(contentRoot, relPath)
	createDummyImage(t, fullPath, 100, 100)

	// Initialize Resizer
	resizer := NewResizer(contentRoot, cacheRoot)

	// Test Resize
	width := 50
	cachedPath, err := resizer.Resize(relPath, width)
	if err != nil {
		t.Fatalf("Resize failed: %v", err)
	}

	// Verify cached file exists
	if _, err := os.Stat(cachedPath); os.IsNotExist(err) {
		t.Errorf("Cached file not created at %s", cachedPath)
	}

	// Verify dimensions
	f, err := os.Open(cachedPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != width {
		t.Errorf("Expected width %d, got %d", width, cfg.Width)
	}
}

func createDummyImage(t *testing.T, path string, w, h int) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Fill with some color
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}
