package middleware_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avijitnpm/modular-monolith/internal/middleware"
)

func TestBodyLimit_UnderLimit_Succeeds(t *testing.T) {
	handler := middleware.BodyLimit(1024)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	body := bytes.NewReader(make([]byte, 512))
	req := httptest.NewRequest(http.MethodPost, "/", body)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rr.Code)
	}
}

func TestBodyLimit_OverLimit_Returns413(t *testing.T) {
	handler := middleware.BodyLimit(1024)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	body := bytes.NewReader(make([]byte, 2048))
	req := httptest.NewRequest(http.MethodPost, "/", body)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %d, want 413", rr.Code)
	}
}
