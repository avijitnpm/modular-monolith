package health

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestReady_LogsErrorWhenUnhealthy(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	h := NewHandler(&mockPinger{err: errors.New("connection refused")}, WithLogger(logger))
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rr := httptest.NewRecorder()
	h.Ready(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", rr.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "readiness check failed") {
		t.Fatalf("expected error log, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "postgres") {
		t.Fatalf("expected dependency name in log, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "connection refused") {
		t.Fatalf("expected error detail in log, got: %s", logOutput)
	}
}

func TestReady_NoErrorLogWhenHealthy(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	h := NewHandler(&mockPinger{}, WithLogger(logger))
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rr := httptest.NewRecorder()
	h.Ready(rr, req)

	if buf.Len() != 0 {
		t.Fatalf("expected no error log when healthy, got: %s", buf.String())
	}
}
