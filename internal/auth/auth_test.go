package auth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mustAuth builds an Auth pinned to fixed Google endpoints and a fixed clock.
func mustAuth(t *testing.T, tokenURL, userInfoURL string) *Auth {
	t.Helper()
	a, err := New(Config{
		ClientID:      "cid",
		ClientSecret:  "secret",
		RedirectURL:   "https://api.merlmartin.com/auth/callback",
		AllowedEmails: []string{"Merl@bennusystems.com"}, // mixed case on purpose
		SessionKey:    []byte("k"),
		SuccessURL:    "https://merlmartin.com/capture/",
		CookieDomain:  "merlmartin.com",
		AuthURL:       "https://accounts.example/auth",
		TokenURL:      tokenURL,
		UserInfoURL:   userInfoURL,
		Now:           func() time.Time { return time.Unix(1000, 0) },
	})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestNewRequiresClientAndKey(t *testing.T) {
	if _, err := New(Config{ClientID: "c"}); err == nil {
		t.Fatal("expected error without secret and session key")
	}
}

func TestLoginRedirectsAndSetsState(t *testing.T) {
	a := mustAuth(t, "http://token.invalid", "http://userinfo.invalid")
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/login", nil))

	if w.Code != http.StatusFound {
		t.Fatalf("code = %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.HasPrefix(loc, "https://accounts.example/auth?") ||
		!strings.Contains(loc, "client_id=cid") || !strings.Contains(loc, "state=") {
		t.Fatalf("location = %s", loc)
	}
	var stateSet bool
	for _, c := range w.Result().Cookies() {
		if c.Name == stateCookie && c.Value != "" && c.HttpOnly {
			stateSet = true
		}
	}
	if !stateSet {
		t.Fatal("expected an HttpOnly state cookie")
	}
}

func googleStub(t *testing.T, email string, verified bool) (tokenURL, userInfoURL string, close func()) {
	t.Helper()
	token := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"access_token":"at","token_type":"Bearer"}`)
	}))
	info := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer at" {
			t.Errorf("userinfo auth = %q, want Bearer at", r.Header.Get("Authorization"))
		}
		v := "false"
		if verified {
			v = "true"
		}
		io.WriteString(w, `{"email":"`+email+`","email_verified":`+v+`}`)
	}))
	return token.URL, info.URL, func() { token.Close(); info.Close() }
}

func callbackReq(state string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/auth/callback?code=xyz&state="+state, nil)
	r.AddCookie(&http.Cookie{Name: stateCookie, Value: "s123"})
	return r
}

func TestCallbackHappyPathIssuesUsableSession(t *testing.T) {
	tokenURL, infoURL, done := googleStub(t, "merl@bennusystems.com", true)
	defer done()
	a := mustAuth(t, tokenURL, infoURL)

	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, callbackReq("s123"))

	if w.Code != http.StatusFound {
		t.Fatalf("code = %d body = %s", w.Code, w.Body.String())
	}
	if loc := w.Header().Get("Location"); loc != "https://merlmartin.com/capture/" {
		t.Fatalf("location = %s", loc)
	}
	var session *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			session = c
		}
	}
	if session == nil || !session.HttpOnly || !session.Secure {
		t.Fatalf("session cookie = %+v", session)
	}
	// The issued cookie must authenticate a follow-up request.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(session)
	if email, ok := a.Current(req); !ok || email != "merl@bennusystems.com" {
		t.Fatalf("Current = %q, %v", email, ok)
	}
}

func TestCallbackRejectsStateMismatch(t *testing.T) {
	a := mustAuth(t, "http://token.invalid", "http://userinfo.invalid")
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, callbackReq("evil")) // cookie is s123, query is evil
	if w.Code != http.StatusForbidden {
		t.Fatalf("code = %d, want 403", w.Code)
	}
}

func TestCallbackRejectsDisallowedEmail(t *testing.T) {
	tokenURL, infoURL, done := googleStub(t, "intruder@gmail.com", true)
	defer done()
	a := mustAuth(t, tokenURL, infoURL)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, callbackReq("s123"))
	if w.Code != http.StatusForbidden {
		t.Fatalf("code = %d, want 403 for disallowed email", w.Code)
	}
}

func TestCallbackRejectsUnverifiedEmail(t *testing.T) {
	tokenURL, infoURL, done := googleStub(t, "merl@bennusystems.com", false)
	defer done()
	a := mustAuth(t, tokenURL, infoURL)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, callbackReq("s123"))
	if w.Code != http.StatusForbidden {
		t.Fatalf("code = %d, want 403 for unverified email", w.Code)
	}
}

func TestRequireSessionGate(t *testing.T) {
	a := mustAuth(t, "http://token.invalid", "http://userinfo.invalid")
	protected := a.RequireSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "secret")
	}))

	w := httptest.NewRecorder()
	protected.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no session: code = %d, want 401", w.Code)
	}

	value, _ := Sign([]byte("k"), Session{Email: "merl@bennusystems.com", Exp: time.Unix(1000, 0).Add(time.Hour).Unix()})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: value})
	w = httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Body.String() != "secret" {
		t.Fatalf("valid session: code = %d body = %q", w.Code, w.Body.String())
	}
}
