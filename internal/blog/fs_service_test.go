package blog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"personalwebsite/internal/blog"
)

func TestFilesystemService_GetAllPosts(t *testing.T) {
	// 1. Create temporary directory with a sample markdown file
	tmpDir := t.TempDir()

	content := []byte(`---
title: "Test Post"
date: "2023-10-27"
summary: "This is a test summary."
---

# Hello World
This is the content.`)

	fileName := "test-post.md"
	filePath := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// 2. Initialize NewFilesystemService pointing to that directory
	// This function does not exist yet, so this test will fail to compile (Red phase)
	service := blog.NewFilesystemService(tmpDir)

	// 3. Assert the returned post has correct Title, Date, Summary, and Slug
	posts, err := service.GetAllPosts()
	if err != nil {
		t.Fatalf("GetAllPosts returned error: %v", err)
	}

	if len(posts) != 1 {
		t.Fatalf("Expected 1 post, got %d", len(posts))
	}

	post := posts[0]

	if post.Title != "Test Post" {
		t.Errorf("Expected Title 'Test Post', got '%s'", post.Title)
	}

	// We expect the date to be parsed. Allowing for some flexibility on time/zone
	// but generally expecting the date components to match.
	expectedDateStr := "2023-10-27"
	if post.Date.Format("2006-01-02") != expectedDateStr {
		t.Errorf("Expected Date %s, got %s", expectedDateStr, post.Date.Format("2006-01-02"))
	}

	if post.Summary != "This is a test summary." {
		t.Errorf("Expected Summary 'This is a test summary.', got '%s'", post.Summary)
	}

	if post.Slug != "test-post" {
		t.Errorf("Expected Slug 'test-post', got '%s'", post.Slug)
	}

	// 4. Assert the Content is rendered HTML
	if !strings.Contains(post.Content, "<h1>") && !strings.Contains(post.Content, "<h1 ") {
		t.Errorf("Expected Content to contain <h1> tag, got: %s", post.Content)
	}

	if !strings.Contains(post.Content, "Hello World") {
		t.Errorf("Expected Content to contain 'Hello World', got: %s", post.Content)
	}
}

func TestFilesystemService_LinkedPhotos(t *testing.T) {
	tmpDir := t.TempDir()

	content := []byte(`---
title: "Photo Post"
date: "2023-10-28"
summary: "Post with photos."
linked_photos:
  - "/images/photo1.jpg"
  - "/images/photo2.jpg"
---

# Content`)

	fileName := "photo-post.md"
	filePath := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	service := blog.NewFilesystemService(tmpDir)

	posts, err := service.GetAllPosts()
	if err != nil {
		t.Fatalf("GetAllPosts returned error: %v", err)
	}

	if len(posts) != 1 {
		t.Fatalf("Expected 1 post, got %d", len(posts))
	}

	post := posts[0]

	if len(post.LinkedPhotos) != 2 {
		t.Fatalf("Expected 2 linked photos, got %d", len(post.LinkedPhotos))
	}

	if post.LinkedPhotos[0] != "/images/photo1.jpg" {
		t.Errorf("Expected first photo '/images/photo1.jpg', got '%s'", post.LinkedPhotos[0])
	}
}
