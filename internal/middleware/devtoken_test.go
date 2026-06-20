package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestDevTokenGating verifies the conditional registration pattern used for /api/v1/token.
// In production, the route is not registered, resulting in 404/405.

func TestDevTokenGating_DevelopmentExposesEndpoint(t *testing.T) {
	env := "development"
	r := chi.NewRouter()
	r.Route("/api/v1", func(api chi.Router) {
		if env == "development" {
			api.Get("/token", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/token", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("development: got %d, want 200", rr.Code)
	}
}

func TestDevTokenGating_ProductionBlocksEndpoint(t *testing.T) {
	env := "production"
	r := chi.NewRouter()
	r.Route("/api/v1", func(api chi.Router) {
		if env == "development" {
			api.Get("/token", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/token", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code == http.StatusOK {
		t.Fatalf("production: got 200, want non-200 (404/405)")
	}
}
