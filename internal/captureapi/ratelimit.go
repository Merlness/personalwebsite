package captureapi

import (
	"math"
	"net/http"
	"sync"
	"time"
)

// rateLimiter is a token bucket: it starts full with `burst` tokens and
// refills at `refillPerSec`. It bounds the /agent (LLM + write) and mutating
// proxy paths so a single leaked app token cannot drive unbounded Anthropic
// spend or GitHub writes. The Anthropic monthly cap stays the external
// backstop; this is the in-process ceiling.
type rateLimiter struct {
	mu           sync.Mutex
	tokens       float64
	burst        float64
	refillPerSec float64
	last         time.Time
	now          func() time.Time
}

func newRateLimiter(burst int, refillPerMin float64, now func() time.Time) *rateLimiter {
	if now == nil {
		now = time.Now
	}
	return &rateLimiter{
		tokens:       float64(burst),
		burst:        float64(burst),
		refillPerSec: refillPerMin / 60,
		last:         now(),
		now:          now,
	}
}

// allow reports whether a request may proceed, consuming one token if so.
func (l *rateLimiter) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.tokens = math.Min(l.burst, l.tokens+now.Sub(l.last).Seconds()*l.refillPerSec)
	l.last = now
	if l.tokens >= 1 {
		l.tokens--
		return true
	}
	return false
}

func rateLimited(l *rateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow() {
			w.Header().Set("Retry-After", "2")
			http.Error(w, `{"message":"rate limit exceeded, slow down"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func orInt(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func orFloat(v, def float64) float64 {
	if v <= 0 {
		return def
	}
	return v
}
