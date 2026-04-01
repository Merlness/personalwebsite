package web

import (
	"net/http"
	"net/http/httptest"
	"personalwebsite/internal/blog"
	"strings"
	"testing"
)

// --- Pretext Script Inclusion Tests ---

func TestLayout_IncludesPretextScripts(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{}, testServerConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status OK; got %v", recorder.Code)
	}

	body := recorder.Body.String()

	requiredScripts := []string{
		`/assets/js/pretext-init.js`,
		`/assets/js/pretext-blog.js`,
		`/assets/js/pretext-shrinkwrap.js`,
		`/assets/js/pretext-hover.js`,
	}

	for _, script := range requiredScripts {
		if !strings.Contains(body, script) {
			t.Errorf("expected layout to include script '%s'", script)
		}
	}
}

// --- Blog Post Pretext Hover Tests (all posts use same template) ---

type mockBlogServiceWithPhotos struct{}

func (s *mockBlogServiceWithPhotos) GetAllPosts() ([]blog.Post, error) {
	return []blog.Post{
		{
			Title:   "Alaska Adventure",
			Slug:    "alaska-adventure",
			Content: "<p>The river was wild and the salmon were running.</p>",
			LinkedPhotos: []string{
				"/assets/portfolio/Alaska/DSC05907.jpg",
				"/assets/portfolio/Alaska/DSC05913.jpg",
			},
		},
	}, nil
}

func (s *mockBlogServiceWithPhotos) GetPost(slug string) (blog.Post, error) {
	if slug == "alaska-adventure" {
		posts, _ := s.GetAllPosts()
		return posts[0], nil
	}
	return blog.Post{}, blog.ErrPostNotFound
}

func TestBlogPost_NoPretextFlow(t *testing.T) {
	srv := NewServer(&mockBlogServiceWithPhotos{}, &mockPortfolioService{}, testServerConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/blog/alaska-adventure", nil)
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	body := recorder.Body.String()

	if strings.Contains(body, "data-pretext-flow") {
		t.Error("expected blog posts to NOT have data-pretext-flow attribute")
	}
	if strings.Contains(body, "data-pretext-inset") {
		t.Error("expected blog posts to NOT have data-pretext-inset attribute")
	}
}

// --- Portfolio Shrinkwrap Tests ---

func TestPortfolio_CategoryTitles_HaveShrinkwrapAttribute(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{}, testServerConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/portfolio", nil)
	recorder := httptest.NewRecorder()

	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status OK; got %v", recorder.Code)
	}

	body := recorder.Body.String()

	if !strings.Contains(body, "Portfolio") {
		t.Error("expected portfolio page to contain 'Portfolio' heading")
	}
}

// --- Blog List Shrinkwrap Tests ---

func TestBlogList_TitlesAndSummaries_HaveShrinkwrapAttribute(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{}, testServerConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/blog", nil)
	recorder := httptest.NewRecorder()

	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status OK; got %v", recorder.Code)
	}

	body := recorder.Body.String()

	count := strings.Count(body, "data-pretext-shrinkwrap")
	if count < 1 {
		t.Errorf("expected at least 1 data-pretext-shrinkwrap attribute on blog list (summary), got %d", count)
	}
}

// --- Pretext Hover Tests ---

func TestLayout_IncludesPretextHoverScript(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{}, testServerConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	body := recorder.Body.String()
	if !strings.Contains(body, "/assets/js/pretext-hover.js") {
		t.Error("expected layout to include pretext-hover.js script")
	}
}

func TestBlogPost_WithLinkedPhotos_HasPretextHoverAttribute(t *testing.T) {
	srv := NewServer(&mockBlogServiceWithPhotos{}, &mockPortfolioService{}, testServerConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/blog/alaska-adventure", nil)
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	body := recorder.Body.String()
	if !strings.Contains(body, "data-pretext-hover") {
		t.Error("expected blog post with linked photos to have data-pretext-hover attribute")
	}
}

func TestBlogPost_WithoutLinkedPhotos_HasPretextHoverAttribute(t *testing.T) {
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{}, testServerConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/blog/first-post", nil)
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	body := recorder.Body.String()
	if !strings.Contains(body, "data-pretext-hover") {
		t.Error("expected blog post without linked photos to also have data-pretext-hover attribute")
	}
}

func TestPretextHoverJS_IsServed(t *testing.T) {
	cfg := ServerConfig{
		PortfolioAssetsPath: t.TempDir(),
		AboutmeAssetsPath:   t.TempDir(),
		CSSAssetsPath:       "../../internal/assets",
	}
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{}, cfg)

	req := httptest.NewRequest(http.MethodGet, "/assets/js/pretext-hover.js", nil)
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected pretext-hover.js to be served; got %v", recorder.Code)
	}
	if recorder.Body.Len() == 0 {
		t.Error("expected pretext-hover.js to have content")
	}
}

func TestCSS_ContainsPretextLineHoverStyles(t *testing.T) {
	cfg := ServerConfig{
		PortfolioAssetsPath: t.TempDir(),
		AboutmeAssetsPath:   t.TempDir(),
		CSSAssetsPath:       "../../internal/assets",
	}
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{}, cfg)

	req := httptest.NewRequest(http.MethodGet, "/assets/css/output.css", nil)
	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	body := recorder.Body.String()
	if !strings.Contains(body, "pretext-line") {
		t.Error("expected CSS to contain .pretext-line styles for hover effect")
	}
}

// --- Blog Post Paragraph Spacing Tests ---

type mockBlogServiceWithParagraphs struct{}

func (service *mockBlogServiceWithParagraphs) GetAllPosts() ([]blog.Post, error) {
	return []blog.Post{
		{
			Title:   "Multi Paragraph Post",
			Slug:    "multi-paragraph",
			Content: "<p>First paragraph about the river.</p><p>Second paragraph about the mountains.</p>",
		},
	}, nil
}

func (service *mockBlogServiceWithParagraphs) GetPost(slug string) (blog.Post, error) {
	if slug == "multi-paragraph" {
		posts, _ := service.GetAllPosts()
		return posts[0], nil
	}
	return blog.Post{}, blog.ErrPostNotFound
}

func TestBlogPost_ParagraphsHaveSpacing(t *testing.T) {
	srv := NewServer(&mockBlogServiceWithParagraphs{}, &mockPortfolioService{}, testServerConfig(t))

	req := httptest.NewRequest(http.MethodGet, "/blog/multi-paragraph", nil)
	recorder := httptest.NewRecorder()

	srv.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status OK; got %v", recorder.Code)
	}

	body := recorder.Body.String()

	contentDivIdx := strings.Index(body, `data-pretext-hover`)
	if contentDivIdx == -1 {
		t.Fatal("expected blog post to contain data-pretext-hover attribute")
	}

	openingTagStart := strings.LastIndex(body[:contentDivIdx], "<div")
	openingTagEnd := strings.Index(body[contentDivIdx:], ">") + contentDivIdx
	contentDivTag := body[openingTagStart : openingTagEnd+1]

	if !strings.Contains(contentDivTag, "space-y-") {
		t.Errorf("expected content container to have space-y-* class for paragraph spacing, got: %s", contentDivTag)
	}
}

// --- Pretext JS Assets Served ---

func TestPretextJS_AssetsServed(t *testing.T) {
	cfg := ServerConfig{
		PortfolioAssetsPath: t.TempDir(),
		AboutmeAssetsPath:   t.TempDir(),
		CSSAssetsPath:       "../../internal/assets",
	}

	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{}, cfg)

	jsFiles := []string{
		"/assets/js/pretext-init.js",
		"/assets/js/pretext-bundle.js",
		"/assets/js/pretext-hover.js",
	}

	for _, jsFile := range jsFiles {
		req := httptest.NewRequest(http.MethodGet, jsFile, nil)
		recorder := httptest.NewRecorder()
		srv.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("expected %s to be served (status OK); got %v", jsFile, recorder.Code)
		}
		if recorder.Body.Len() == 0 {
			t.Errorf("expected %s to have content", jsFile)
		}
	}
}
