package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avijitnpm/modular-monolith/internal/middleware"
)

var metricsNext = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestMetricsAuth_MissingToken_Returns401(t *testing.T) {
	handler := middleware.MetricsAuth("secret-token", metricsNext)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", rr.Code)
	}
}

func TestMetricsAuth_InvalidToken_Returns401(t *testing.T) {
	handler := middleware.MetricsAuth("secret-token", metricsNext)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", rr.Code)
	}
}

func TestMetricsAuth_ValidToken_Returns200(t *testing.T) {
	handler := middleware.MetricsAuth("secret-token", metricsNext)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rr.Code)
	}
}

func TestMetricsAuth_EmptyConfig_AllowsAll(t *testing.T) {
	handler := middleware.MetricsAuth("", metricsNext)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rr.Code)
	}
}
