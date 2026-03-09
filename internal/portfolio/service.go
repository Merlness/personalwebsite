package portfolio

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ErrCategoryNotFound = errors.New("category not found")

type Image struct {
	Path string
	Ext  string
}

type Category struct {
	Name       string
	Images     []Image
	CoverImage Image
}

type Service interface {
	GetCategories() ([]Category, error)
	GetCategory(name string) (Category, error)
}

type filesystemService struct {
	root          string
	webPathPrefix string
}

func NewFilesystemService(root, webPathPrefix string) Service {
	return &filesystemService{
		root:          root,
		webPathPrefix: webPathPrefix,
	}
}

func (s *filesystemService) GetCategory(name string) (Category, error) {
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

var preferredOrder = []string{"Landscape", "People", "Wildlife", "Structures"}

func (s *filesystemService) GetCategories() ([]Category, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}

	existingDirs := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			existingDirs[entry.Name()] = true
		}
	}

	var categories []Category

	for _, catName := range preferredOrder {
		if existingDirs[catName] {
			cat, err := s.scanCategory(catName)
			if err != nil {
				return nil, err
			}
			categories = append(categories, cat)
			delete(existingDirs, catName)
		}
	}

	remaining := make([]string, 0, len(existingDirs))
	for dirName := range existingDirs {
		remaining = append(remaining, dirName)
	}
	sort.Strings(remaining)

	for _, catName := range remaining {
		cat, err := s.scanCategory(catName)
		if err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}

	return categories, nil
}

func (s *filesystemService) scanCategory(categoryName string) (Category, error) {
	dirPath := filepath.Join(s.root, categoryName)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return Category{}, err
	}

	var images []Image

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))

		if strings.Contains(name, "_w600") || strings.Contains(name, "_w1600") {
			continue
		}

		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
			baseName := strings.TrimSuffix(name, ext)
			imgPath := filepath.Join(s.webPathPrefix, categoryName, baseName)

			images = append(images, Image{Path: imgPath, Ext: ext})
		}
	}

	sort.Slice(images, func(idx, jdx int) bool {
		return images[idx].Path < images[jdx].Path
	})

	var coverImage Image
	if len(images) > 0 {
		coverImage = images[len(images)-1]
	}

	return Category{
		Name:       categoryName,
		Images:     images,
		CoverImage: coverImage,
	}, nil
}
