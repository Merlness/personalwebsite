package blog

import (
	"testing"
	"time"
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

func TestLinkedCategory_ReturnsAlaskaFromAlaskaPhotos(t *testing.T) {
	post := Post{
		LinkedPhotos: []string{
			"/assets/portfolio/Alaska/DSC05907.jpg",
			"/assets/portfolio/Alaska/DSC05913.jpg",
		},
	}

	category := post.LinkedCategory()
	if category != "Alaska" {
		t.Errorf("Expected 'Alaska', got '%s'", category)
	}
}

func TestLinkedCategory_ReturnsWildlifeFromWildlifePhotos(t *testing.T) {
	post := Post{
		LinkedPhotos: []string{"/assets/portfolio/Wildlife/DSC01260.jpg"},
	}

	category := post.LinkedCategory()
	if category != "Wildlife" {
		t.Errorf("Expected 'Wildlife', got '%s'", category)
	}
}

func TestLinkedCategory_ReturnsEmptyWhenNoPhotos(t *testing.T) {
	post := Post{
		LinkedPhotos: []string{},
	}

	category := post.LinkedCategory()
	if category != "" {
		t.Errorf("Expected empty string, got '%s'", category)
	}
}

func TestFindNeighbors_MiddlePost(t *testing.T) {
	posts := []Post{
		{Slug: "newest", Date: date(2018, 7, 4)},
		{Slug: "middle", Date: date(2018, 6, 23)},
		{Slug: "oldest", Date: date(2018, 6, 19)},
	}

	prevPost, nextPost := FindNeighbors(posts, "middle")

	if prevPost == nil || prevPost.Slug != "newest" {
		t.Errorf("Expected prev to be 'newest', got %v", prevPost)
	}
	if nextPost == nil || nextPost.Slug != "oldest" {
		t.Errorf("Expected next to be 'oldest', got %v", nextPost)
	}
}

func TestFindNeighbors_FirstPost(t *testing.T) {
	posts := []Post{
		{Slug: "newest", Date: date(2018, 7, 4)},
		{Slug: "oldest", Date: date(2018, 6, 19)},
	}

	prevPost, nextPost := FindNeighbors(posts, "newest")

	if prevPost != nil {
		t.Errorf("Expected prev to be nil for first post, got %v", prevPost)
	}
	if nextPost == nil || nextPost.Slug != "oldest" {
		t.Errorf("Expected next to be 'oldest', got %v", nextPost)
	}
}

func TestFindNeighbors_LastPost(t *testing.T) {
	posts := []Post{
		{Slug: "newest", Date: date(2018, 7, 4)},
		{Slug: "oldest", Date: date(2018, 6, 19)},
	}

	prevPost, nextPost := FindNeighbors(posts, "oldest")

	if prevPost == nil || prevPost.Slug != "newest" {
		t.Errorf("Expected prev to be 'newest', got %v", prevPost)
	}
	if nextPost != nil {
		t.Errorf("Expected next to be nil for last post, got %v", nextPost)
	}
}

func TestFindNeighbors_NotFound(t *testing.T) {
	posts := []Post{
		{Slug: "only", Date: date(2018, 7, 4)},
	}

	prevPost, nextPost := FindNeighbors(posts, "nonexistent")

	if prevPost != nil {
		t.Errorf("Expected prev to be nil, got %v", prevPost)
	}
	if nextPost != nil {
		t.Errorf("Expected next to be nil, got %v", nextPost)
	}
}

func date(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}
