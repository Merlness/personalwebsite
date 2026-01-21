package blog

import (
	"errors"
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
