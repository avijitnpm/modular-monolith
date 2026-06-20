package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(ctx context.Context) error {
	return m.err
}

func TestLive_Returns200(t *testing.T) {
	h := NewHandler(&mockPinger{})
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rr := httptest.NewRecorder()
	h.Live(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rr.Code)
	}
}

func TestLive_ReturnsCorrectPayload(t *testing.T) {
	h := NewHandler(&mockPinger{})
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rr := httptest.NewRecorder()
	h.Live(rr, req)

	var body map[string]string
	json.NewDecoder(rr.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Fatalf("got status %q, want %q", body["status"], "ok")
	}
}

func TestReady_Returns200WhenDBHealthy(t *testing.T) {
	h := NewHandler(&mockPinger{})
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rr := httptest.NewRecorder()
	h.Ready(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rr.Code)
	}
}

func TestReady_Returns503WhenDBUnhealthy(t *testing.T) {
	h := NewHandler(&mockPinger{err: errors.New("connection refused")})
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rr := httptest.NewRecorder()
	h.Ready(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", rr.Code)
	}
}

func TestReady_ReturnsCorrectPayloadWhenHealthy(t *testing.T) {
	h := NewHandler(&mockPinger{})
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rr := httptest.NewRecorder()
	h.Ready(rr, req)

	var body map[string]string
	json.NewDecoder(rr.Body).Decode(&body)
	if body["status"] != "ready" {
		t.Fatalf("got status %q, want %q", body["status"], "ready")
	}
}

func TestReady_ReturnsCorrectPayloadWhenUnhealthy(t *testing.T) {
	h := NewHandler(&mockPinger{err: errors.New("timeout")})
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rr := httptest.NewRecorder()
	h.Ready(rr, req)

	var body map[string]string
	json.NewDecoder(rr.Body).Decode(&body)
	if body["status"] != "not_ready" {
		t.Fatalf("got status %q, want %q", body["status"], "not_ready")
	}
}
