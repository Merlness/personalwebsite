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

// --- BuildPhotoToBlogMap tests ---

func TestBuildPhotoToBlogMap_MapsPhotosToSlugs(t *testing.T) {
	posts := []Post{
		{
			Slug: "alaska-trip",
			LinkedPhotos: []string{
				"/assets/portfolio/Alaska/DSC05907.jpg",
				"/assets/portfolio/Alaska/DSC05913.png",
			},
		},
	}

	m := BuildPhotoToBlogMap(posts)

	if m["/assets/portfolio/Alaska/DSC05907"] != "alaska-trip" {
		t.Errorf("Expected DSC05907 mapped to 'alaska-trip', got '%s'", m["/assets/portfolio/Alaska/DSC05907"])
	}
	if m["/assets/portfolio/Alaska/DSC05913"] != "alaska-trip" {
		t.Errorf("Expected DSC05913 mapped to 'alaska-trip', got '%s'", m["/assets/portfolio/Alaska/DSC05913"])
	}
}

func TestBuildPhotoToBlogMap_EmptyPosts(t *testing.T) {
	m := BuildPhotoToBlogMap([]Post{})

	if len(m) != 0 {
		t.Errorf("Expected empty map for empty posts, got %d entries", len(m))
	}
}

func TestBuildPhotoToBlogMap_NilLinkedPhotos(t *testing.T) {
	posts := []Post{
		{Slug: "no-photos", LinkedPhotos: nil},
	}

	m := BuildPhotoToBlogMap(posts)

	if len(m) != 0 {
		t.Errorf("Expected empty map when post has nil LinkedPhotos, got %d entries", len(m))
	}
}

func TestBuildPhotoToBlogMap_MultiplePostsDifferentCategories(t *testing.T) {
	posts := []Post{
		{
			Slug:         "alaska-trip",
			LinkedPhotos: []string{"/assets/portfolio/Alaska/DSC05907.jpg"},
		},
		{
			Slug:         "wildlife-watch",
			LinkedPhotos: []string{"/assets/portfolio/Wildlife/DSC01260.jpg"},
		},
	}

	m := BuildPhotoToBlogMap(posts)

	if len(m) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(m))
	}
	if m["/assets/portfolio/Alaska/DSC05907"] != "alaska-trip" {
		t.Errorf("Expected Alaska photo mapped to 'alaska-trip', got '%s'", m["/assets/portfolio/Alaska/DSC05907"])
	}
	if m["/assets/portfolio/Wildlife/DSC01260"] != "wildlife-watch" {
		t.Errorf("Expected Wildlife photo mapped to 'wildlife-watch', got '%s'", m["/assets/portfolio/Wildlife/DSC01260"])
	}
}

func TestBuildPhotoToBlogMap_DuplicatePhoto_LastPostWins(t *testing.T) {
	posts := []Post{
		{
			Slug:         "first-post",
			LinkedPhotos: []string{"/assets/portfolio/Alaska/DSC05907.jpg"},
		},
		{
			Slug:         "second-post",
			LinkedPhotos: []string{"/assets/portfolio/Alaska/DSC05907.jpg"},
		},
	}

	m := BuildPhotoToBlogMap(posts)

	// The last post in the slice should win since it overwrites the map key
	if m["/assets/portfolio/Alaska/DSC05907"] != "second-post" {
		t.Errorf("Expected duplicate photo mapped to last post 'second-post', got '%s'", m["/assets/portfolio/Alaska/DSC05907"])
	}
}

func TestBuildPhotoToBlogMap_StripsExtensionCorrectly(t *testing.T) {
	posts := []Post{
		{
			Slug: "test-post",
			LinkedPhotos: []string{
				"/assets/portfolio/Cat/photo.jpg",
				"/assets/portfolio/Cat/image.jpeg",
				"/assets/portfolio/Cat/pic.png",
			},
		},
	}

	m := BuildPhotoToBlogMap(posts)

	expected := map[string]string{
		"/assets/portfolio/Cat/photo": "test-post",
		"/assets/portfolio/Cat/image": "test-post",
		"/assets/portfolio/Cat/pic":   "test-post",
	}

	for key, expectedSlug := range expected {
		if m[key] != expectedSlug {
			t.Errorf("Key '%s': expected '%s', got '%s'", key, expectedSlug, m[key])
		}
	}
}

// --- LinkedCategory edge case ---

func TestLinkedCategory_MalformedPath_ReturnsEmpty(t *testing.T) {
	tests := []struct {
		name         string
		linkedPhotos []string
		expected     string
	}{
		{"no slashes", []string{"photo.jpg"}, ""},
		{"one segment", []string{"/photo.jpg"}, ""},
		{"two segments", []string{"/assets/photo.jpg"}, ""},
		{"three segments", []string{"/assets/portfolio/photo.jpg"}, ""},
		{"nil photos", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := Post{LinkedPhotos: tt.linkedPhotos}
			got := post.LinkedCategory()
			if got != tt.expected {
				t.Errorf("LinkedCategory() = '%s', want '%s'", got, tt.expected)
			}
		})
	}
}

func date(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}
