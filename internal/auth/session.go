// Package auth provides Google sign-in for the organizer PWA: an OAuth
// authorization-code flow that issues a signed, HttpOnly session cookie so
// the phone never holds a GitHub or app token. It is enabled only when the
// Google client and session key are configured; otherwise the app-token path
// stays in force.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Session is the signed payload carried in the cookie.
type Session struct {
	Email string `json:"email"`
	Exp   int64  `json:"exp"`
}

// Sign returns a tamper-proof cookie value: base64url(payload).base64url(hmac).
func Sign(key []byte, s Session) (string, error) {
	payload, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(payload)
	return body + "." + mac(key, body), nil
}

// Verify checks the signature and expiry and returns the session.
func Verify(key []byte, value string, now time.Time) (Session, error) {
	body, sig, ok := strings.Cut(value, ".")
	if !ok {
		return Session{}, fmt.Errorf("malformed session")
	}
	if !hmac.Equal([]byte(sig), []byte(mac(key, body))) {
		return Session{}, fmt.Errorf("bad signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return Session{}, fmt.Errorf("bad encoding: %w", err)
	}
	var s Session
	if err := json.Unmarshal(raw, &s); err != nil {
		return Session{}, fmt.Errorf("bad payload: %w", err)
	}
	if now.Unix() > s.Exp {
		return Session{}, fmt.Errorf("expired")
	}
	return s, nil
}

func mac(key []byte, msg string) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(msg))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
