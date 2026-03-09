package blog

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/yuin/goldmark"
)

type filesystemService struct {
	dir string
}

func NewFilesystemService(dir string) Service {
	return &filesystemService{dir: dir}
}

func parsePost(filePath string) (Post, error) {
	fileContent, readErr := os.ReadFile(filePath)
	if readErr != nil {
		return Post{}, readErr
	}

	var meta struct {
		Title        string   `yaml:"title"`
		Date         string   `yaml:"date"`
		Summary      string   `yaml:"summary"`
		LinkedPhotos []string `yaml:"linked_photos"`
	}

	rest, parseErr := frontmatter.Parse(bytes.NewReader(fileContent), &meta)
	if parseErr != nil {
		return Post{}, parseErr
	}

	var buf bytes.Buffer
	if convertErr := goldmark.Convert(rest, &buf); convertErr != nil {
		return Post{}, convertErr
	}

	date, dateErr := time.Parse("2006-01-02", meta.Date)
	if dateErr != nil {
		return Post{}, dateErr
	}

	fileName := filepath.Base(filePath)
	slug := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	return Post{
		Title:        meta.Title,
		Slug:         slug,
		Date:         date,
		Summary:      meta.Summary,
		Content:      buf.String(),
		LinkedPhotos: meta.LinkedPhotos,
	}, nil
}

func (svc *filesystemService) GetAllPosts() ([]Post, error) {
	entries, readErr := os.ReadDir(svc.dir)
	if readErr != nil {
		return nil, readErr
	}

	var posts []Post
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		entryPath := filepath.Join(svc.dir, entry.Name())
		post, parseErr := parsePost(entryPath)
		if parseErr != nil {
			return nil, parseErr
		}

		posts = append(posts, post)
	}

	return posts, nil
}

func (svc *filesystemService) GetPost(slug string) (Post, error) {
	postPath := filepath.Join(svc.dir, slug+".md")

	if _, statErr := os.Stat(postPath); os.IsNotExist(statErr) {
		return Post{}, ErrPostNotFound
	}

	post, parseErr := parsePost(postPath)
	if parseErr != nil {
		return Post{}, parseErr
	}

	return post, nil
}
