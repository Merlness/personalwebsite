package images

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

type Resizer struct {
	contentRoot string
	cacheRoot   string
}

func NewResizer(contentRoot, cacheRoot string) *Resizer {
	return &Resizer{
		contentRoot: contentRoot,
		cacheRoot:   cacheRoot,
	}
}

func (r *Resizer) Resize(relPath string, width int) (string, error) {
	fullPath := filepath.Join(r.contentRoot, relPath)

	// Check if source exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("source file not found: %w", err)
	}

	// Construct cache path
	dir := filepath.Dir(relPath)
	filename := filepath.Base(relPath)
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	cachedFilename := fmt.Sprintf("%s_w%d%s", nameWithoutExt, width, ext)
	cachedDir := filepath.Join(r.cacheRoot, dir)
	cachedPath := filepath.Join(cachedDir, cachedFilename)

	// Check if cached exists
	if _, err := os.Stat(cachedPath); err == nil {
		return cachedPath, nil
	}

	// Ensure cache dir exists
	if err := os.MkdirAll(cachedDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache dir: %w", err)
	}

	// Resize and save
	srcImage, err := imaging.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to open image: %w", err)
	}

	// Resize to width, preserving aspect ratio (height = 0)
	dstImage := imaging.Resize(srcImage, width, 0, imaging.Lanczos)

	if err := imaging.Save(dstImage, cachedPath); err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}

	return cachedPath, nil
}
