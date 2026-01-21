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

func (s *filesystemService) GetAllPosts() ([]Post, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	var posts []Post
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(s.dir, entry.Name())
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		var meta struct {
			Title        string   `yaml:"title"`
			Date         string   `yaml:"date"`
			Summary      string   `yaml:"summary"`
			LinkedPhotos []string `yaml:"linked_photos"`
		}

		rest, err := frontmatter.Parse(bytes.NewReader(fileContent), &meta)
		if err != nil {
			return nil, err
		}

		var buf bytes.Buffer
		if err := goldmark.Convert(rest, &buf); err != nil {
			return nil, err
		}

		date, err := time.Parse("2006-01-02", meta.Date)
		if err != nil {
			// If date parsing fails, maybe just default or skip?
			// For now let's error as the requirements are specific
			return nil, err
		}

		slug := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))

		posts = append(posts, Post{
			Title:        meta.Title,
			Slug:         slug,
			Date:         date,
			Summary:      meta.Summary,
			Content:      buf.String(),
			LinkedPhotos: meta.LinkedPhotos,
		})
	}

	return posts, nil
}

func (s *filesystemService) GetPost(slug string) (Post, error) {
	// Simple implementation reusing GetAllPosts
	posts, err := s.GetAllPosts()
	if err != nil {
		return Post{}, err
	}
	for _, p := range posts {
		if p.Slug == slug {
			return p, nil
		}
	}
	return Post{}, ErrPostNotFound
}
