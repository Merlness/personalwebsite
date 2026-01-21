package web

import (
	"os"
	"strings"
	"testing"
)

func TestRHCPThemeCSS(t *testing.T) {
	// Path to input.css relative to internal/web
	cssPath := "../assets/css/input.css"
	content, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("Failed to read css file: %v", err)
	}

	css := string(content)

	// Find the rhcp theme block
	if !strings.Contains(css, `[data-theme="rhcp"]`) {
		t.Fatalf("rhcp theme block not found in css")
	}

	// Extract the block (simple approximation)
	start := strings.Index(css, `[data-theme="rhcp"]`)
	end := start + strings.Index(css[start:], "}")
	block := css[start:end]

	// Requirements for the variable block
	varChecks := []struct {
		name     string
		subStr   string
		required bool
	}{
		// Background color: Dark Black
		{"Background Color", "--color-bg-primary: #0a0a0a", true},
		// Text Color: Light Silver
		{"Text Color", "--color-text-primary: #e5e7eb", true},
		// Heading Color: Magenta
		{"Heading Color", "--color-heading: #D500F9", true},
		// Accent Color: Cyan
		{"Accent Color", "--color-accent: #00E5FF", true},
		// Background Pattern presence
		{"Background Pattern", "background-image: url('data:image/svg+xml;base64,", true},
	}

	for _, check := range varChecks {
		if check.required && !strings.Contains(block, check.subStr) {
			t.Errorf("RHCP theme variable block missing %s: expected to contain '%s'", check.name, check.subStr)
		}
	}

	// Global requirements (can be outside the variable block)
	globalChecks := []struct {
		name     string
		subStr   string
		required bool
	}{
		// Border decoration on main
		{"Border Decoration", "border-image", true},
		// Hero decoration
		{"Spicy Decoration", ".spicy-decoration", true},
	}

	for _, check := range globalChecks {
		if check.required && !strings.Contains(css, check.subStr) {
			t.Errorf("CSS missing %s: expected to contain '%s'", check.name, check.subStr)
		}
	}
}
