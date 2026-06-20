package middleware

import (
	"net"
	"net/http"
	"sync"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/pkg/response"
	"golang.org/x/time/rate"
)

type rateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
}

func newRateLimiter(r rate.Limit, burst int) *rateLimiter {
	return &rateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    burst,
	}
}

func (rl *rateLimiter) get(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	l, exists := rl.limiters[key]
	if !exists {
		l = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[key] = l
	}
	return l
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// PublicRateLimit limits anonymous endpoints to 10 requests/minute keyed by IP.
func PublicRateLimit() func(http.Handler) http.Handler {
	rl := newRateLimiter(rate.Limit(10.0/60.0), 10)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.get(clientIP(r)).Allow() {
				response.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AuthenticatedRateLimit limits authenticated mutations to 60 requests/minute keyed by user ID.
func AuthenticatedRateLimit() func(http.Handler) http.Handler {
	rl := newRateLimiter(rate.Limit(60.0/60.0), 60)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientIP(r)
			if user, ok := appcontext.GetAuthenticatedUser(r.Context()); ok && user.UserID != "" {
				key = user.UserID
			}
			if !rl.get(key).Allow() {
				response.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// WebhookRateLimit limits webhook ingress to 120 requests/minute keyed by IP.
func WebhookRateLimit() func(http.Handler) http.Handler {
	rl := newRateLimiter(rate.Limit(120.0/60.0), 120)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.get(clientIP(r)).Allow() {
				response.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
