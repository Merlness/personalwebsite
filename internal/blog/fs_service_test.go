package blog_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"personalwebsite/internal/blog"
)

func writeMarkdownFile(t *testing.T, dir, slug, title, date, summary, body string) {
	t.Helper()
	content := fmt.Sprintf(`---
title: "%s"
date: "%s"
summary: "%s"
---

%s`, title, date, summary, body)
	filePath := filepath.Join(dir, slug+".md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file %s: %v", filePath, err)
	}
}

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

func TestFilesystemService_GetAllPosts_ReturnsMostRecentFirst(t *testing.T) {
	tmpDir := t.TempDir()
	writeMarkdownFile(t, tmpDir, "oldest", "Oldest Post", "2018-06-19", "The oldest.", "# Old")
	writeMarkdownFile(t, tmpDir, "middle", "Middle Post", "2018-06-23", "The middle.", "# Mid")
	writeMarkdownFile(t, tmpDir, "newest", "Newest Post", "2018-07-04", "The newest.", "# New")

	service := blog.NewFilesystemService(tmpDir)

	posts, err := service.GetAllPosts()
	if err != nil {
		t.Fatalf("GetAllPosts returned error: %v", err)
	}

	if len(posts) != 3 {
		t.Fatalf("Expected 3 posts, got %d", len(posts))
	}

	if posts[0].Slug != "newest" {
		t.Errorf("Expected first post to be 'newest', got '%s'", posts[0].Slug)
	}

	if posts[1].Slug != "middle" {
		t.Errorf("Expected second post to be 'middle', got '%s'", posts[1].Slug)
	}

	if posts[2].Slug != "oldest" {
		t.Errorf("Expected third post to be 'oldest', got '%s'", posts[2].Slug)
	}
}

func TestFilesystemService_GetPost_ReturnsCorrectPost(t *testing.T) {
	tmpDir := t.TempDir()
	writeMarkdownFile(t, tmpDir, "my-trip", "My Trip", "2024-03-15", "A great adventure.", "# My Trip\nIt was amazing.")

	service := blog.NewFilesystemService(tmpDir)

	post, err := service.GetPost("my-trip")
	if err != nil {
		t.Fatalf("GetPost returned error: %v", err)
	}

	if post.Title != "My Trip" {
		t.Errorf("Expected Title 'My Trip', got '%s'", post.Title)
	}

	if post.Date.Format("2006-01-02") != "2024-03-15" {
		t.Errorf("Expected Date '2024-03-15', got '%s'", post.Date.Format("2006-01-02"))
	}

	if post.Summary != "A great adventure." {
		t.Errorf("Expected Summary 'A great adventure.', got '%s'", post.Summary)
	}

	if post.Slug != "my-trip" {
		t.Errorf("Expected Slug 'my-trip', got '%s'", post.Slug)
	}

	if !strings.Contains(post.Content, "My Trip") {
		t.Errorf("Expected Content to contain 'My Trip', got: %s", post.Content)
	}
}

func TestFilesystemService_GetPost_ReturnsErrPostNotFoundForNonexistentSlug(t *testing.T) {
	tmpDir := t.TempDir()
	writeMarkdownFile(t, tmpDir, "existing-post", "Existing", "2024-01-01", "Exists.", "# Exists")

	service := blog.NewFilesystemService(tmpDir)

	_, err := service.GetPost("nonexistent")
	if !errors.Is(err, blog.ErrPostNotFound) {
		t.Errorf("Expected ErrPostNotFound, got: %v", err)
	}
}

func TestFilesystemService_GetPost_ReturnsCorrectPostAmongMultiple(t *testing.T) {
	tmpDir := t.TempDir()
	writeMarkdownFile(t, tmpDir, "alpha-post", "Alpha", "2024-01-01", "First post.", "# Alpha")
	writeMarkdownFile(t, tmpDir, "beta-post", "Beta", "2024-02-02", "Second post.", "# Beta")
	writeMarkdownFile(t, tmpDir, "gamma-post", "Gamma", "2024-03-03", "Third post.", "# Gamma")

	service := blog.NewFilesystemService(tmpDir)

	post, err := service.GetPost("beta-post")
	if err != nil {
		t.Fatalf("GetPost returned error: %v", err)
	}

	if post.Title != "Beta" {
		t.Errorf("Expected Title 'Beta', got '%s'", post.Title)
	}

	if post.Slug != "beta-post" {
		t.Errorf("Expected Slug 'beta-post', got '%s'", post.Slug)
	}

	if post.Summary != "Second post." {
		t.Errorf("Expected Summary 'Second post.', got '%s'", post.Summary)
	}
}

func TestFilesystemService_GetPost_SucceedsEvenWhenOtherFilesAreInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	writeMarkdownFile(t, tmpDir, "good-post", "Good Post", "2024-05-01", "A valid post.", "# Good content")

	invalidContent := []byte(`---
title: "Bad Post"
date: "not-a-real-date"
summary: "This will fail to parse."
---

# Bad`)
	invalidPath := filepath.Join(tmpDir, "bad-post.md")
	if err := os.WriteFile(invalidPath, invalidContent, 0644); err != nil {
		t.Fatalf("Failed to write invalid test file: %v", err)
	}

	service := blog.NewFilesystemService(tmpDir)

	post, err := service.GetPost("good-post")
	if err != nil {
		t.Fatalf("GetPost should succeed for valid post even when other files are invalid, got error: %v", err)
	}

	if post.Title != "Good Post" {
		t.Errorf("Expected Title 'Good Post', got '%s'", post.Title)
	}

	if post.Slug != "good-post" {
		t.Errorf("Expected Slug 'good-post', got '%s'", post.Slug)
	}
}
