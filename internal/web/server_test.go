package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"personalwebsite/internal/blog"
	"personalwebsite/internal/portfolio"
	"strings"
	"testing"
)

func testServerConfig(t *testing.T) ServerConfig {
	t.Helper()
	return ServerConfig{
		PortfolioAssetsPath: t.TempDir(),
		AboutmeAssetsPath:   t.TempDir(),
		CSSAssetsPath:       "../../internal/assets",
	}
}

type mockPortfolioService struct{}

func (s *mockPortfolioService) GetCategories() ([]portfolio.Category, error) {
	return []portfolio.Category{
		{Name: "Landscape", Images: []portfolio.Image{{Path: "/assets/l", Ext: ".jpg"}}, CoverImage: portfolio.Image{Path: "/assets/l", Ext: ".jpg"}},
		{Name: "Wildlife", Images: []portfolio.Image{{Path: "/assets/w", Ext: ".jpg"}}, CoverImage: portfolio.Image{Path: "/assets/w", Ext: ".jpg"}},
		{Name: "Portraits", Images: []portfolio.Image{{Path: "/assets/p", Ext: ".jpg"}}, CoverImage: portfolio.Image{Path: "/assets/p", Ext: ".jpg"}},
	}, nil
}

func (s *mockPortfolioService) GetCategory(name string) (portfolio.Category, error) {
	if name == "Landscape" {
		return portfolio.Category{
			Name:       "Landscape",
			Images:     []portfolio.Image{{Path: "/assets/l", Ext: ".jpg"}},
			CoverImage: portfolio.Image{Path: "/assets/l", Ext: ".jpg"},
		}, nil
	}
	return portfolio.Category{}, portfolio.ErrCategoryNotFound
}

func TestServer(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{}, testServerConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), "<title>Personal Website</title>") {
		t.Errorf("expected title 'Personal Website'; got body: %s", recorder.Body.String())
	}
}

func TestBlogPost(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{}, testServerConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/blog/first-post", nil)
	recorder := httptest.NewRecorder()

	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", recorder.Code)
	}

	body := recorder.Body.String()
	required := []string{"My First Post", "Here is the full content"}
	for _, expected := range required {
		if !strings.Contains(body, expected) {
			t.Errorf("expected body to contain '%s'; got body: %s", expected, body)
		}
	}
}

func TestCSSAssets(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{}, testServerConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/assets/css/output.css", nil)
	recorder := httptest.NewRecorder()

	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", recorder.Code)
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "text/css" {
		t.Errorf("expected content-type text/css; got %s", contentType)
	}
}

func TestAbout(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{}, testServerConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	recorder := httptest.NewRecorder()

	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", recorder.Code)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "<title>About Me | Merl Martin</title>") {
		t.Errorf("expected title 'About Me | Merl Martin'; got body: %s", body)
	}

	if !strings.Contains(body, "Contact") {
		t.Errorf("expected body to contain 'Contact'; got body: %s", body)
	}

	expectedImage := "spicy.jpg"
	if !strings.Contains(body, expectedImage) {
		t.Errorf("expected body to contain image '%s'; got body snippet: %s", expectedImage, body[:200])
	}
}

func TestPortfolio(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{}, testServerConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/portfolio", nil)
	recorder := httptest.NewRecorder()

	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", recorder.Code)
	}

	body := recorder.Body.String()
	expectedCategories := []string{"Landscape", "Wildlife", "Portraits"}
	for _, cat := range expectedCategories {
		if !strings.Contains(body, cat) {
			t.Errorf("expected body to contain category '%s'; got body: %s", cat, body)
		}
	}
}

func TestPortfolioAssets(t *testing.T) {
	portfolioDir := t.TempDir()

	testContent := "test content"
	testFilePath := filepath.Join(portfolioDir, "test.txt")
	if err := os.WriteFile(testFilePath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg := ServerConfig{
		PortfolioAssetsPath: portfolioDir,
		AboutmeAssetsPath:   t.TempDir(),
		CSSAssetsPath:       "../../internal/assets",
	}

	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{}, cfg)

	req := httptest.NewRequest(http.MethodGet, "/assets/portfolio/test.txt", nil)
	recorder := httptest.NewRecorder()

	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", recorder.Code)
	}

	body := recorder.Body.String()
	if strings.TrimSpace(body) != testContent {
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
	srv := NewServer(&mockLinkedPhotosService{}, &mockPortfolioService{}, testServerConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/blog/photo-post", nil)
	recorder := httptest.NewRecorder()

	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", recorder.Code)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "View in Portfolio") && !strings.Contains(body, "Related Collection") {
		t.Errorf("expected body to contain 'View in Portfolio' or 'Related Collection'; got body: %s", body)
	}
}

func TestPortfolio_LinkedStory(t *testing.T) {
	blogSvc := &mockLinkedPhotosService{}
	portSvc := &mockPortfolioServiceWithPhoto{}

	srv := NewServer(blogSvc, portSvc, testServerConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/portfolio/TestCat", nil)
	recorder := httptest.NewRecorder()

	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", recorder.Code)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, `"/images/p1":"photo-post"`) {
		t.Errorf("expected body to contain photo mapping; got body: %s", body)
	}
}

func TestPortfolioCategory(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{}, testServerConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/portfolio/Landscape", nil)
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status OK; got %v", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "Landscape") {
		t.Errorf("expected body to contain 'Landscape'")
	}

	req = httptest.NewRequest(http.MethodGet, "/portfolio/Invalid", nil)
	recorder = httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Errorf("expected status NotFound; got %v", recorder.Code)
	}
}

type mockPortfolioServiceWithPhoto struct{}

func (s *mockPortfolioServiceWithPhoto) GetCategories() ([]portfolio.Category, error) {
	return []portfolio.Category{
		{Name: "TestCat", Images: []portfolio.Image{{Path: "/images/p1", Ext: ".jpg"}}},
	}, nil
}

func (s *mockPortfolioServiceWithPhoto) GetCategory(name string) (portfolio.Category, error) {
	if name == "TestCat" {
		return portfolio.Category{Name: "TestCat", Images: []portfolio.Image{{Path: "/images/p1", Ext: ".jpg"}}}, nil
	}
	return portfolio.Category{}, portfolio.ErrCategoryNotFound
}
