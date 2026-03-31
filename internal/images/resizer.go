package images

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

	srcInfo, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("source file not found: %w", err)
	}

	dir := filepath.Dir(relPath)
	filename := filepath.Base(relPath)
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	cachedFilename := fmt.Sprintf("%s_w%d%s", nameWithoutExt, width, ext)
	cachedDir := filepath.Join(r.cacheRoot, dir)
	cachedPath := filepath.Join(cachedDir, cachedFilename)

	cachedInfo, cacheErr := os.Stat(cachedPath)
	if cacheErr == nil && cachedInfo.ModTime().After(srcInfo.ModTime()) {
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

	// Atomic write: save to temp file first, then rename.
	// This prevents race conditions where concurrent requests could
	// corrupt the cache file by writing to it simultaneously.
	cachedExt := filepath.Ext(cachedPath)
	cachedBase := strings.TrimSuffix(cachedPath, cachedExt)
	tmpPath := cachedBase + ".tmp." + strconv.Itoa(os.Getpid()) + cachedExt
	if err := imaging.Save(dstImage, tmpPath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to save image: %w", err)
	}
	if err := os.Rename(tmpPath, cachedPath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to move cache file: %w", err)
	}

	return cachedPath, nil
}
