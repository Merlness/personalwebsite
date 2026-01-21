package web

import (
	"net/http"
	"net/http/httptest"
	"personalwebsite/internal/blog"
	"strings"
	"testing"
)

func TestThemeSwitcher(t *testing.T) {
	// Setup server with mock services
	srv := NewServer(blog.NewMemoryService(), &mockPortfolioService{})

	// Request the home page
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status OK; got %v", w.Code)
	}

	body := w.Body.String()

	// Check for Alpine.js theme data initialization
	// We expect something that handles theme state
	if !strings.Contains(body, "x-data") || !strings.Contains(body, "theme") {
		t.Errorf("expected Alpine.js theme initialization in body")
	}

	// Check for the presence of a theme toggle button
	if !strings.Contains(body, "Toggle Theme") && !strings.Contains(body, "Switch Theme") {
		// Maybe it's an icon, but we should probably look for the button element with a specific click handler
		if !strings.Contains(body, "@click") {
			// This is a weak check, but let's start with expecting some Alpine click handler
			t.Errorf("expected theme toggle button with @click handler")
		}
	}

	// Check if the html tag has the dynamic class or data attribute binding
	if !strings.Contains(body, ":data-theme") && !strings.Contains(body, "x-bind:data-theme") && !strings.Contains(body, "x-effect") {
		t.Errorf("expected html tag to have dynamic theme binding")
	}
}
