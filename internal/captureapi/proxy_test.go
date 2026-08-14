package captureapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testConfig(upstream string) Config {
	return Config{
		AppToken:       "app-secret",
		GitHubToken:    "gh-secret",
		Owner:          "Merlness",
		Repo:           "life-organizer",
		AllowedOrigins: []string{"https://merlmartin.com"},
		Upstream:       upstream,
	}
}

func do(h http.Handler, method, path, token string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestRejectsMissingOrWrongToken(t *testing.T) {
	h := NewHandler(testConfig("http://unused.invalid"))
	for name, token := range map[string]string{"missing": "", "wrong": "nope"} {
		t.Run(name, func(t *testing.T) {
			w := do(h, http.MethodGet, "/repos/Merlness/life-organizer/contents/tasks.md?ref=main", token, "")
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("got %d, want 401", w.Code)
			}
		})
	}
}

func TestRejectsOtherReposAndTraversal(t *testing.T) {
	h := NewHandler(testConfig("http://unused.invalid"))
	for name, path := range map[string]string{
		"other repo": "/repos/Merlness/personalwebsite/contents/go.mod?ref=main",
		"other user": "/repos/evil/life-organizer/contents/x?ref=main",
		"traversal":  "/repos/Merlness/life-organizer/contents/../../secrets?ref=main",
	} {
		t.Run(name, func(t *testing.T) {
			w := do(h, http.MethodGet, path, "app-secret", "")
			if w.Code != http.StatusForbidden {
				t.Fatalf("got %d, want 403", w.Code)
			}
		})
	}
}

func TestForwardsGetWithServerSideGitHubToken(t *testing.T) {
	var seenAuth, seenPath, seenQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		seenPath = r.URL.Path
		seenQuery = r.URL.RawQuery
		io.WriteString(w, `{"content":"aGk=","sha":"abc"}`)
	}))
	defer srv.Close()

	h := NewHandler(testConfig(srv.URL))
	w := do(h, http.MethodGet, "/repos/Merlness/life-organizer/contents/tasks.md?ref=main", "app-secret", "")

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	if seenAuth != "Bearer gh-secret" {
		t.Fatalf("upstream auth = %q, want server-side GitHub token", seenAuth)
	}
	if seenPath != "/repos/Merlness/life-organizer/contents/tasks.md" || seenQuery != "ref=main" {
		t.Fatalf("upstream got %s?%s", seenPath, seenQuery)
	}
	if !strings.Contains(w.Body.String(), `"sha":"abc"`) {
		t.Fatalf("body not passed through: %s", w.Body.String())
	}
}

func TestForwardsPutBodyAndUpstreamStatus(t *testing.T) {
	var seenBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusConflict) // sha mismatch passes through for client retry
		io.WriteString(w, `{"message":"conflict"}`)
	}))
	defer srv.Close()

	h := NewHandler(testConfig(srv.URL))
	payload := `{"message":"m","content":"aGk=","sha":"old","branch":"main"}`
	w := do(h, http.MethodPut, "/repos/Merlness/life-organizer/contents/tasks.md", "app-secret", payload)

	if w.Code != http.StatusConflict {
		t.Fatalf("got %d, want upstream 409 passed through", w.Code)
	}
	if string(seenBody) != payload {
		t.Fatalf("upstream body = %s", seenBody)
	}
}

func TestRejectsPostMethod(t *testing.T) {
	h := NewHandler(testConfig("http://unused.invalid"))
	w := do(h, http.MethodPost, "/repos/Merlness/life-organizer/contents/tasks.md", "app-secret", "{}")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got %d, want 405", w.Code)
	}
}

func TestCORSPreflightAndEcho(t *testing.T) {
	h := NewHandler(testConfig("http://unused.invalid"))

	req := httptest.NewRequest(http.MethodOptions, "/repos/Merlness/life-organizer/contents/tasks.md", nil)
	req.Header.Set("Origin", "https://merlmartin.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("preflight got %d, want 204", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://merlmartin.com" {
		t.Fatalf("allow-origin = %q", got)
	}

	req = httptest.NewRequest(http.MethodOptions, "/repos/Merlness/life-organizer/contents/tasks.md", nil)
	req.Header.Set("Origin", "https://evil.example")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected allow-origin for foreign origin: %q", got)
	}
}

func TestHealthzNeedsNoAuth(t *testing.T) {
	h := NewHandler(testConfig("http://unused.invalid"))
	w := do(h, http.MethodGet, "/healthz", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
}

func TestAgentMountRequiresAppToken(t *testing.T) {
	cfg := testConfig("http://unused.invalid")
	cfg.Agent = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"reply":"ok"}`))
	})
	h := NewHandler(cfg)

	if w := do(h, http.MethodPost, "/agent", "", `{"message":"x"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("no token: got %d, want 401", w.Code)
	}
	if w := do(h, http.MethodPost, "/agent", "wrong", `{"message":"x"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token: got %d, want 401", w.Code)
	}
	w := do(h, http.MethodPost, "/agent", "app-secret", `{"message":"x"}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "ok") {
		t.Fatalf("good token: got %d %s", w.Code, w.Body.String())
	}
}

func TestAgentAbsentWhenNotConfigured(t *testing.T) {
	h := NewHandler(testConfig("http://unused.invalid"))
	if w := do(h, http.MethodPost, "/agent", "app-secret", `{}`); w.Code == http.StatusOK {
		t.Fatalf("unmounted /agent should not serve 200, got %d", w.Code)
	}
}

func TestSessionAuthorizesWithoutAppToken(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"content":"eA==","sha":"s"}`))
	}))
	defer upstream.Close()

	cfg := testConfig(upstream.URL)
	cfg.Agent = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"reply":"ok"}`)) })
	cfg.Session = func(r *http.Request) (string, bool) {
		c, err := r.Cookie("sess")
		return "merl@bennusystems.com", err == nil && c.Value == "good"
	}
	h := NewHandler(cfg)

	signedIn := func(method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(""))
		req.AddCookie(&http.Cookie{Name: "sess", Value: "good"})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	if w := signedIn(http.MethodGet, "/repos/Merlness/life-organizer/contents/tasks.md?ref=main"); w.Code != http.StatusOK {
		t.Fatalf("files with session: got %d, want 200", w.Code)
	}
	if w := signedIn(http.MethodPost, "/agent"); w.Code != http.StatusOK {
		t.Fatalf("agent with session: got %d, want 200", w.Code)
	}

	// A bad session must not fall through to unauthenticated access.
	req := httptest.NewRequest(http.MethodGet, "/repos/Merlness/life-organizer/contents/tasks.md?ref=main", nil)
	req.AddCookie(&http.Cookie{Name: "sess", Value: "stale"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("stale session: got %d, want 401", w.Code)
	}

	// The app token still works, so the phone keeps running mid-migration.
	if w := do(h, http.MethodGet, "/repos/Merlness/life-organizer/contents/tasks.md?ref=main", "app-secret", ""); w.Code != http.StatusOK {
		t.Fatalf("app token: got %d, want 200", w.Code)
	}
}

func TestCORSAllowsCredentialsForAllowedOrigin(t *testing.T) {
	h := NewHandler(testConfig("http://unused.invalid"))
	req := httptest.NewRequest(http.MethodOptions, "/agent", nil)
	req.Header.Set("Origin", "https://merlmartin.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allow-credentials = %q, want true", got)
	}
	if !strings.Contains(w.Header().Get("Access-Control-Allow-Methods"), "POST") {
		t.Fatalf("allow-methods = %q, want POST", w.Header().Get("Access-Control-Allow-Methods"))
	}

	// An unknown origin must get no CORS grant at all.
	req = httptest.NewRequest(http.MethodOptions, "/agent", nil)
	req.Header.Set("Origin", "https://evil.test")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Header().Get("Access-Control-Allow-Credentials") != "" || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("CORS granted to an origin that is not on the allowlist")
	}
}
