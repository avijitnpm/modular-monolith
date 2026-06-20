package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/pkg/response"
	"golang.org/x/time/rate"
)

const (
	cleanupInterval = 3 * time.Minute
	staleTimeout    = 3 * time.Minute
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     rate.Limit
	burst    int
}

func newRateLimiter(r rate.Limit, burst int) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     r,
		burst:    burst,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) get(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[key]
	if !exists {
		v = &visitor{limiter: rate.NewLimiter(rl.rate, rl.burst)}
		rl.visitors[key] = v
	}
	v.lastSeen = time.Now()
	return v.limiter
}

func (rl *rateLimiter) cleanup() {
	for {
		time.Sleep(cleanupInterval)
		rl.mu.Lock()
		for key, v := range rl.visitors {
			if time.Since(v.lastSeen) > staleTimeout {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

func clientIP(r *http.Request) string {
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
