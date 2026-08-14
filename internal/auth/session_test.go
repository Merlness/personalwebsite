package auth

import (
	"testing"
	"time"
)

func TestSignVerifyRoundTrip(t *testing.T) {
	key := []byte("test-key")
	now := time.Unix(1000, 0)
	v, err := Sign(key, Session{Email: "merl@bennusystems.com", Exp: now.Add(time.Hour).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	s, err := Verify(key, v, now)
	if err != nil {
		t.Fatal(err)
	}
	if s.Email != "merl@bennusystems.com" {
		t.Fatalf("email = %q", s.Email)
	}
}

func TestVerifyRejectsExpired(t *testing.T) {
	key := []byte("k")
	v, _ := Sign(key, Session{Email: "x", Exp: time.Unix(1000, 0).Unix()})
	if _, err := Verify(key, v, time.Unix(2000, 0)); err == nil {
		t.Fatal("expected expired error")
	}
}

func TestVerifyRejectsTamperAndWrongKey(t *testing.T) {
	key := []byte("k")
	v, _ := Sign(key, Session{Email: "x", Exp: time.Unix(9999999999, 0).Unix()})
	if _, err := Verify(key, "A"+v[1:], time.Unix(1000, 0)); err == nil {
		t.Fatal("expected signature error on tampered payload")
	}
	if _, err := Verify([]byte("other-key"), v, time.Unix(1000, 0)); err == nil {
		t.Fatal("expected signature error on wrong key")
	}
}
