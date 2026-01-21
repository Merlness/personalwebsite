package blog

import (
	"testing"
)

func TestNewMemoryService(t *testing.T) {
	svc := NewMemoryService()

	posts, err := svc.GetAllPosts()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(posts) == 0 {
		t.Error("expected at least one post, got 0")
	}
}
