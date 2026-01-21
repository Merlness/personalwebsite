package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"personalwebsite/internal/blog"
	"personalwebsite/internal/portfolio"
	"strings"
	"testing"
)

type mockPortfolioService struct{}

func (s *mockPortfolioService) GetCategories() ([]portfolio.Category, error) {
	return []portfolio.Category{
		{Name: "Landscape", Images: []string{"/assets/l.jpg"}},
		{Name: "Wildlife", Images: []string{"/assets/w.jpg"}},
		{Name: "Portraits", Images: []string{"/assets/p.jpg"}},
	}, nil
}

func (s *mockPortfolioService) GetCategory(name string) (portfolio.Category, error) {
	if name == "Landscape" {
		return portfolio.Category{Name: "Landscape", Images: []string{"/assets/l.jpg"}}, nil
	}
	return portfolio.Category{}, portfolio.ErrCategoryNotFound
}

func TestServer(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", w.Code)
	}

	if !strings.Contains(w.Body.String(), "<title>Personal Website</title>") {
		t.Errorf("expected title 'Personal Website'; got body: %s", w.Body.String())
	}
}

func TestBlogPost(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{})

	req := httptest.NewRequest(http.MethodGet, "/blog/first-post", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", w.Code)
	}

	body := w.Body.String()
	required := []string{"My First Post", "Here is the full content"}
	for _, s := range required {
		if !strings.Contains(body, s) {
			t.Errorf("expected body to contain '%s'; got body: %s", s, body)
		}
	}
}

func TestCSSAssets(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{})

	req := httptest.NewRequest(http.MethodGet, "/assets/css/output.css", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/css" {
		t.Errorf("expected content-type text/css; got %s", contentType)
	}
}

func TestAbout(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{})

	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "<title>About Me | Merl Martin</title>") {
		t.Errorf("expected title 'About Me | Merl Martin'; got body: %s", body)
	}

	if !strings.Contains(body, "Contact") {
		t.Errorf("expected body to contain 'Contact'; got body: %s", body)
	}

	// Check for the new image URL
	expectedImage := "spicy.jpg"
	if !strings.Contains(body, expectedImage) {
		t.Errorf("expected body to contain image '%s'; got body snippet: %s", expectedImage, body[:200]) // snippet to avoid huge log
	}
}

func TestPortfolio(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{})

	req := httptest.NewRequest(http.MethodGet, "/portfolio", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", w.Code)
	}

	body := w.Body.String()
	categories := []string{"Landscape", "Wildlife", "Portraits"}
	for _, cat := range categories {
		if !strings.Contains(body, cat) {
			t.Errorf("expected body to contain category '%s'; got body: %s", cat, body)
		}
	}
}

func TestPortfolioAssets(t *testing.T) {
	// Setup: Create a temporary file in content/portfolio
	// We need to handle the path relative to where tests run (internal/web)
	contentDir := "../../content/portfolio"
	_, err := os.Stat(contentDir)
	if err != nil {
		// If running from root or elsewhere, try to find it
		fallback := "content/portfolio"
		if _, err2 := os.Stat(fallback); err2 == nil {
			contentDir = fallback
		} else {
			cwd, _ := os.Getwd()
			t.Fatalf("Could not find content/portfolio directory.\nTried: %s (err: %v)\nTried: %s (err: %v)\nCurrent dir: %s", contentDir, err, fallback, err2, cwd)
		}
	}

	testFile := "test.txt"
	fullPath := contentDir + "/" + testFile
	err = os.WriteFile(fullPath, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(fullPath)

	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{})

	// Test serving from content/portfolio
	req := httptest.NewRequest(http.MethodGet, "/assets/portfolio/test.txt", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", w.Code)
	}

	body := w.Body.String()
	if strings.TrimSpace(body) != "test content" {
		t.Errorf("expected body 'test content'; got '%s'", body)
	}
}

type mockLinkedPhotosService struct{}

func (s *mockLinkedPhotosService) GetAllPosts() ([]blog.Post, error) {
	return []blog.Post{
		{
			Title:        "Photo Post",
			Slug:         "photo-post",
			Content:      "Content",
			LinkedPhotos: []string{"/images/p1.jpg"},
		},
	}, nil
}

func (s *mockLinkedPhotosService) GetPost(slug string) (blog.Post, error) {
	if slug == "photo-post" {
		return blog.Post{
			Title:        "Photo Post",
			Slug:         "photo-post",
			Content:      "Content",
			LinkedPhotos: []string{"/images/p1.jpg"},
		}, nil
	}
	return blog.Post{}, blog.ErrPostNotFound
}

func TestBlogPost_LinkedPhotos(t *testing.T) {
	srv := NewServer(&mockLinkedPhotosService{}, &mockPortfolioService{})

	req := httptest.NewRequest(http.MethodGet, "/blog/photo-post", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "View in Portfolio") && !strings.Contains(body, "Related Collection") {
		t.Errorf("expected body to contain 'View in Portfolio' or 'Related Collection'; got body: %s", body)
	}
}

func TestPortfolio_LinkedStory(t *testing.T) {
	// Setup services
	blogSvc := &mockLinkedPhotosService{} // Returns photo-post with /images/p1.jpg

	// We need portfolio service to return /images/p1.jpg
	// I'll create a specific mock for this test
	portSvc := &mockPortfolioServiceWithPhoto{}

	srv := NewServer(blogSvc, portSvc)

	// Update: check /portfolio/TestCat instead of /portfolio
	req := httptest.NewRequest(http.MethodGet, "/portfolio/TestCat", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", w.Code)
	}

	body := w.Body.String()
	// We expect the photoToBlog map to be present in the JS
	// The map should contain "/images/p1.jpg": "photo-post"
	// Since it's JSON encoded, it might look like: "/images/p1.jpg":"photo-post"

	if !strings.Contains(body, `"/images/p1.jpg":"photo-post"`) {
		t.Errorf("expected body to contain photo mapping; got body: %s", body)
	}
}

func TestPortfolioCategory(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{})

	// Test valid category
	req := httptest.NewRequest(http.MethodGet, "/portfolio/Landscape", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Landscape") {
		t.Errorf("expected body to contain 'Landscape'")
	}

	// Test invalid category
	req = httptest.NewRequest(http.MethodGet, "/portfolio/Invalid", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status NotFound; got %v", w.Code)
	}
}

type mockPortfolioServiceWithPhoto struct{}

func (s *mockPortfolioServiceWithPhoto) GetCategories() ([]portfolio.Category, error) {
	return []portfolio.Category{
		{Name: "TestCat", Images: []string{"/images/p1.jpg"}},
	}, nil
}

func (s *mockPortfolioServiceWithPhoto) GetCategory(name string) (portfolio.Category, error) {
	if name == "TestCat" {
		return portfolio.Category{Name: "TestCat", Images: []string{"/images/p1.jpg"}}, nil
	}
	return portfolio.Category{}, portfolio.ErrCategoryNotFound
}
