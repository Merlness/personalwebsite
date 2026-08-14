package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	sessionCookie = "merl_session"
	stateCookie   = "merl_oauth_state"
)

// Config configures Google sign-in. The endpoint URLs default to Google's and
// are overridable in tests.
type Config struct {
	ClientID      string
	ClientSecret  string
	RedirectURL   string   // https://api.merlmartin.com/auth/callback
	AllowedEmails []string // the only accounts allowed in
	SessionKey    []byte   // HMAC key for the session cookie
	SuccessURL    string   // where to land after login, e.g. https://merlmartin.com/capture/
	CookieDomain  string   // e.g. merlmartin.com, so the cookie reaches the api subdomain
	SessionTTL    time.Duration

	AuthURL     string
	TokenURL    string
	UserInfoURL string
	HTTP        *http.Client
	Now         func() time.Time
}

// Auth holds a configured sign-in flow.
type Auth struct {
	cfg     Config
	allowed map[string]bool
}

// New fills defaults and lowercases the allowlist. It returns an error if the
// required Google client / session key are missing, so main can leave auth off.
func New(cfg Config) (*Auth, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" || len(cfg.SessionKey) == 0 {
		return nil, fmt.Errorf("auth: ClientID, ClientSecret and SessionKey are required")
	}
	if cfg.AuthURL == "" {
		cfg.AuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
	}
	if cfg.TokenURL == "" {
		cfg.TokenURL = "https://oauth2.googleapis.com/token"
	}
	if cfg.UserInfoURL == "" {
		cfg.UserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"
	}
	if cfg.SuccessURL == "" {
		cfg.SuccessURL = "/"
	}
	if cfg.SessionTTL == 0 {
		cfg.SessionTTL = 180 * 24 * time.Hour
	}
	if cfg.HTTP == nil {
		cfg.HTTP = http.DefaultClient
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	allowed := make(map[string]bool, len(cfg.AllowedEmails))
	for _, e := range cfg.AllowedEmails {
		allowed[strings.ToLower(strings.TrimSpace(e))] = true
	}
	return &Auth{cfg: cfg, allowed: allowed}, nil
}

// Handler mounts /auth/login, /auth/callback, /auth/logout, and /auth/me.
func (a *Auth) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", a.login)
	mux.HandleFunc("/auth/callback", a.callback)
	mux.HandleFunc("/auth/logout", a.logout)
	mux.HandleFunc("/auth/me", a.me)
	return mux
}

// me lets the PWA find out whether this browser is signed in without having
// to provoke a 401 on a real request.
func (a *Auth) me(w http.ResponseWriter, r *http.Request) {
	email, ok := a.Current(r)
	w.Header().Set("Content-Type", "application/json")
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"not signed in"}`))
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"email": email})
}

func (a *Auth) login(w http.ResponseWriter, r *http.Request) {
	state, err := randToken()
	if err != nil {
		http.Error(w, "cannot start sign-in", http.StatusInternalServerError)
		return
	}
	// Short-lived, HttpOnly state cookie backs the CSRF check in callback.
	http.SetCookie(w, a.cookie(stateCookie, state, 10*time.Minute))
	q := url.Values{
		"client_id":     {a.cfg.ClientID},
		"redirect_uri":  {a.cfg.RedirectURL},
		"response_type": {"code"},
		"scope":         {"openid email"},
		"state":         {state},
		"prompt":        {"select_account"},
	}
	http.Redirect(w, r, a.cfg.AuthURL+"?"+q.Encode(), http.StatusFound)
}

func (a *Auth) callback(w http.ResponseWriter, r *http.Request) {
	st, err := r.Cookie(stateCookie)
	if err != nil || st.Value == "" || r.URL.Query().Get("state") != st.Value {
		http.Error(w, "bad oauth state", http.StatusForbidden)
		return
	}
	http.SetCookie(w, a.expire(stateCookie))

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}
	token, err := a.exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "token exchange failed", http.StatusBadGateway)
		return
	}
	email, verified, err := a.userEmail(r.Context(), token)
	if err != nil {
		http.Error(w, "cannot read profile", http.StatusBadGateway)
		return
	}
	if !verified || !a.allowed[strings.ToLower(email)] {
		http.Error(w, "this account is not allowed", http.StatusForbidden)
		return
	}
	value, err := Sign(a.cfg.SessionKey, Session{Email: email, Exp: a.cfg.Now().Add(a.cfg.SessionTTL).Unix()})
	if err != nil {
		http.Error(w, "cannot issue session", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, a.cookie(sessionCookie, value, a.cfg.SessionTTL))
	http.Redirect(w, r, a.cfg.SuccessURL, http.StatusFound)
}

func (a *Auth) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, a.expire(sessionCookie))
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"ok":true}`)
}

// RequireSession gates a handler on a valid session cookie.
func (a *Auth) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := a.Current(r); !ok {
			http.Error(w, `{"message":"sign in required"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Current returns the signed-in email if the request carries a valid session.
func (a *Auth) Current(r *http.Request) (string, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return "", false
	}
	s, err := Verify(a.cfg.SessionKey, c.Value, a.cfg.Now())
	if err != nil {
		return "", false
	}
	return s.Email, true
}

func (a *Auth) exchange(ctx context.Context, code string) (string, error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {a.cfg.ClientID},
		"client_secret": {a.cfg.ClientSecret},
		"redirect_uri":  {a.cfg.RedirectURL},
		"grant_type":    {"authorization_code"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := a.cfg.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint status %d", res.StatusCode)
	}
	var body struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}
	return body.AccessToken, nil
}

func (a *Auth) userEmail(ctx context.Context, accessToken string) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.cfg.UserInfoURL, nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := a.cfg.HTTP.Do(req)
	if err != nil {
		return "", false, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("userinfo status %d", res.StatusCode)
	}
	var body struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return "", false, err
	}
	return body.Email, body.EmailVerified, nil
}

func (a *Auth) cookie(name, value string, ttl time.Duration) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   a.cfg.CookieDomain,
		Expires:  a.cfg.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

func (a *Auth) expire(name string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Domain:   a.cfg.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

func randToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
