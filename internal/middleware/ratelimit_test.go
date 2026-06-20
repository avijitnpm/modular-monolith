package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestPublicRateLimit_AllowsBelowLimit(t *testing.T) {
	mw := PublicRateLimit()(okHandler)

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rr := httptest.NewRecorder()
		mw.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200", i, rr.Code)
		}
	}
}

func TestPublicRateLimit_BlocksAboveLimit(t *testing.T) {
	mw := PublicRateLimit()(okHandler)

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "5.6.7.8:1234"
		rr := httptest.NewRecorder()
		mw.ServeHTTP(rr, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "5.6.7.8:1234"
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", rr.Code)
	}

	var body response.ErrorResponse
	json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "rate limit exceeded" {
		t.Fatalf("got error %q, want %q", body.Error, "rate limit exceeded")
	}
}

func TestPublicRateLimit_KeysByIP(t *testing.T) {
	mw := PublicRateLimit()(okHandler)

	// Exhaust limit for IP A
	for i := 0; i < 11; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rr := httptest.NewRecorder()
		mw.ServeHTTP(rr, req)
	}

	// IP B should still work
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("different IP got %d, want 200", rr.Code)
	}
}

func TestAuthenticatedRateLimit_KeysByUserID(t *testing.T) {
	mw := AuthenticatedRateLimit()(okHandler)

	for i := 0; i < 60; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "1.1.1.1:1234"
		ctx := appcontext.SetAuthenticatedUser(req.Context(), &appcontext.AuthenticatedUser{UserID: "user-a"})
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		mw.ServeHTTP(rr, req)
	}

	// 61st request for user-a should be blocked
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "1.1.1.1:1234"
	ctx := appcontext.SetAuthenticatedUser(req.Context(), &appcontext.AuthenticatedUser{UserID: "user-a"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", rr.Code)
	}
}

func TestAuthenticatedRateLimit_DifferentUsersSeparateBuckets(t *testing.T) {
	mw := AuthenticatedRateLimit()(okHandler)

	// Exhaust user-a
	for i := 0; i < 61; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "2.2.2.2:1234"
		ctx := appcontext.SetAuthenticatedUser(req.Context(), &appcontext.AuthenticatedUser{UserID: "user-a"})
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		mw.ServeHTTP(rr, req)
	}

	// user-b should still work
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "2.2.2.2:1234"
	ctx := appcontext.SetAuthenticatedUser(req.Context(), &appcontext.AuthenticatedUser{UserID: "user-b"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("different user got %d, want 200", rr.Code)
	}
}

func TestAuthenticatedRateLimit_FallsBackToIP(t *testing.T) {
	mw := AuthenticatedRateLimit()(okHandler)

	// No user in context - should key by IP
	for i := 0; i < 60; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "3.3.3.3:1234"
		rr := httptest.NewRecorder()
		mw.ServeHTTP(rr, req)
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "3.3.3.3:1234"
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", rr.Code)
	}
}

func TestWebhookRateLimit_AllowsBelowLimit(t *testing.T) {
	mw := WebhookRateLimit()(okHandler)

	for i := 0; i < 120; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "4.4.4.4:1234"
		rr := httptest.NewRecorder()
		mw.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200", i, rr.Code)
		}
	}
}

func TestWebhookRateLimit_BlocksAboveLimit(t *testing.T) {
	mw := WebhookRateLimit()(okHandler)

	for i := 0; i < 120; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "6.6.6.6:1234"
		rr := httptest.NewRecorder()
		mw.ServeHTTP(rr, req)
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "6.6.6.6:1234"
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", rr.Code)
	}
}

func TestMiddleware_CallsNextHandler(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	mw := PublicRateLimit()(handler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "9.9.9.9:1234"
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if !called {
		t.Fatal("next handler was not called")
	}
}

func TestRateLimiter_EvictsStaleEntries(t *testing.T) {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     1,
		burst:    1,
	}

	rl.get("active")
	rl.get("stale")

	// Manually age the stale entry
	rl.mu.Lock()
	rl.visitors["stale"].lastSeen = time.Now().Add(-4 * time.Minute)
	rl.mu.Unlock()

	// Run cleanup inline
	rl.mu.Lock()
	for key, v := range rl.visitors {
		if time.Since(v.lastSeen) > staleTimeout {
			delete(rl.visitors, key)
		}
	}
	rl.mu.Unlock()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if _, exists := rl.visitors["stale"]; exists {
		t.Fatal("stale entry was not evicted")
	}
	if _, exists := rl.visitors["active"]; !exists {
		t.Fatal("active entry was incorrectly evicted")
	}
}

func TestClientIP_UsesRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:9999"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")

	ip := clientIP(req)
	if ip != "192.168.1.1" {
		t.Fatalf("got %q, want %q", ip, "192.168.1.1")
	}
}
