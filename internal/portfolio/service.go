package portfolio

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ErrCategoryNotFound = errors.New("category not found")

type Category struct {
	Name          string
	Images        []string
	ImageExts     []string // Track extensions for each image
	CoverImage    string
	CoverImageExt string // Track extension for cover
}

type Service interface {
	GetCategories() ([]Category, error)
	GetCategory(name string) (Category, error)
}

type FilesystemService struct {
	root          string
	webPathPrefix string
}

func NewFilesystemService(root, webPathPrefix string) *FilesystemService {
	return &FilesystemService{
		root:          root,
		webPathPrefix: webPathPrefix,
	}
}

func (s *FilesystemService) GetCategory(name string) (Category, error) {
	// Security check: simple check to prevent directory traversal
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return Category{}, ErrCategoryNotFound
	}

	// Check if directory exists
	dirPath := filepath.Join(s.root, name)
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Category{}, ErrCategoryNotFound
		}
		return Category{}, err
	}

	if !info.IsDir() {
		return Category{}, ErrCategoryNotFound
	}

	return s.scanCategory(name)
}

func (s *FilesystemService) GetCategories() ([]Category, error) {
	// Define hardcoded sort order
	order := []string{"Landscape", "People", "Wildlife", "Structures"}

	var categories []Category

	// Read subdirectories of rootPath
	entries, err := os.ReadDir(s.root)
	if err != nil {
		// If root doesn't exist or error, return error
		return nil, err
	}

	// Create a map of existing directories for quick lookup
	existingDirs := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			existingDirs[entry.Name()] = true
		}
	}

	// Iterate based on sort order
	for _, catName := range order {
		if existingDirs[catName] {
			cat, err := s.scanCategory(catName)
			if err != nil {
				return nil, err
			}
			categories = append(categories, cat)
		}
	}

	return categories, nil
}

func (s *FilesystemService) scanCategory(categoryName string) (Category, error) {
	dirPath := filepath.Join(s.root, categoryName)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return Category{}, err
	}

	var images []string
	var exts []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))

		// Skip specialized versions (e.g. _w600.jpg, _w1600.jpg)
		if strings.Contains(name, "_w600") || strings.Contains(name, "_w1600") {
			continue
		}

		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			// Store base path without extension to make template logic cleaner
			baseName := strings.TrimSuffix(name, ext)
			imgPath := filepath.Join(s.webPathPrefix, categoryName, baseName)

			images = append(images, imgPath)
			exts = append(exts, ext)
		}
	}

	// Sort images (and exts in parallel)
	// We need a custom sort here
	type imgWithExt struct {
		path string
		ext  string
	}
	combined := make([]imgWithExt, len(images))
	for i := range images {
		combined[i] = imgWithExt{images[i], exts[i]}
	}
	sort.Slice(combined, func(i, j int) bool {
		return combined[i].path < combined[j].path
	})

	for i := range combined {
		images[i] = combined[i].path
		exts[i] = combined[i].ext
	}

	var coverImage string
	var coverImageExt string
	if len(images) > 0 {
		coverImage = images[len(images)-1]
		coverImageExt = exts[len(exts)-1]
	}

	return Category{
		Name:          categoryName,
		Images:        images,
		ImageExts:     exts,
		CoverImage:    coverImage,
		CoverImageExt: coverImageExt,
	}, nil
}
