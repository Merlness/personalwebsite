package captureapi

import (
	"net/http"
	"testing"
	"time"
)

func TestRateLimiterBurstThenRefill(t *testing.T) {
	now := time.Unix(0, 0)
	clock := func() time.Time { return now }
	l := newRateLimiter(2, 60, clock) // 60/min = 1 token/sec

	if !l.allow() || !l.allow() {
		t.Fatal("a burst of 2 should be allowed")
	}
	if l.allow() {
		t.Fatal("3rd immediate request should be blocked")
	}
	now = now.Add(time.Second) // one token refills
	if !l.allow() {
		t.Fatal("after 1s one token should be available")
	}
	if l.allow() {
		t.Fatal("only one token should have refilled")
	}
}

func TestAgentEndpointRateLimited(t *testing.T) {
	cfg := testConfig("http://unused.invalid")
	cfg.AgentBurst = 2
	cfg.Agent = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"reply":"ok"}`))
	})
	h := NewHandler(cfg)

	var codes []int
	for i := 0; i < 3; i++ {
		codes = append(codes, do(h, http.MethodPost, "/agent", "app-secret", `{"message":"x"}`).Code)
	}
	if codes[0] != http.StatusOK || codes[1] != http.StatusOK || codes[2] != http.StatusTooManyRequests {
		t.Fatalf("codes = %v, want [200 200 429]", codes)
	}
}
