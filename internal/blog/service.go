package blog

import (
	"errors"
	"path/filepath"
	"strings"
	"time"
)

var ErrPostNotFound = errors.New("post not found")

type Post struct {
	Title        string
	Slug         string
	Date         time.Time
	Summary      string
	Content      string // Added content
	LinkedPhotos []string
}

type Service interface {
	GetAllPosts() ([]Post, error)
	GetPost(slug string) (Post, error)
}

type memoryService struct {
	posts []Post
}

func NewMemoryService() Service {
	return &memoryService{
		posts: []Post{
			{
				Title:   "My First Post",
				Slug:    "first-post",
				Date:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				Summary: "This is the summary of my first post.",
				Content: "Here is the full content of my first post. It was a great trip...",
			},
		},
	}
}

func (s *memoryService) GetAllPosts() ([]Post, error) {
	return s.posts, nil
}

func (s *memoryService) GetPost(slug string) (Post, error) {
	for _, p := range s.posts {
		if p.Slug == slug {
			return p, nil
		}
	}
	return Post{}, ErrPostNotFound
}

func (post Post) LinkedCategory() string {
	if len(post.LinkedPhotos) == 0 {
		return ""
	}
	parts := strings.Split(post.LinkedPhotos[0], "/")
	if len(parts) < 4 {
		return ""
	}
	return parts[3]
}

func FindNeighbors(posts []Post, slug string) (*Post, *Post) {
	for idx, post := range posts {
		if post.Slug != slug {
			continue
		}
		var prevPost, nextPost *Post
		if idx > 0 {
			prevPost = &posts[idx-1]
		}
		if idx < len(posts)-1 {
			nextPost = &posts[idx+1]
		}
		return prevPost, nextPost
	}
	return nil, nil
}

func BuildPhotoToBlogMap(posts []Post) map[string]string {
	photoToBlog := make(map[string]string)
	for _, post := range posts {
		for _, photo := range post.LinkedPhotos {
			key := strings.TrimSuffix(photo, filepath.Ext(photo))
			photoToBlog[key] = post.Slug
		}
	}
	return photoToBlog
}
