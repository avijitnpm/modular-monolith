package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetrics_RequestCounterIncrements(t *testing.T) {
	before := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "/test", "200"))

	handler := Metrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	after := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("GET", "/test", "200"))
	if after-before != 1 {
		t.Fatalf("expected counter to increment by 1, got %f", after-before)
	}
}

func TestMetrics_DurationHistogramRecords(t *testing.T) {
	handler := Metrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/duration", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	// Verify that the histogram has at least one observation by checking the metric exists
	count := testutil.CollectAndCount(httpRequestDuration)
	if count == 0 {
		t.Fatal("expected duration histogram to have observations")
	}
}

func TestMetrics_InFlightGauge(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})

	handler := Metrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		<-release
		w.WriteHeader(http.StatusOK)
	}))

	go func() {
		req := httptest.NewRequest(http.MethodGet, "/inflight", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}()

	<-entered
	val := testutil.ToFloat64(httpRequestsInFlight)
	if val < 1 {
		t.Fatalf("expected in-flight >= 1, got %f", val)
	}
	release <- struct{}{}
}

func TestMetrics_ErrorCounterIncrements4xx(t *testing.T) {
	before := testutil.ToFloat64(httpRequestErrorsTotal.WithLabelValues("GET", "/err4", "404"))

	handler := Metrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/err4", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	after := testutil.ToFloat64(httpRequestErrorsTotal.WithLabelValues("GET", "/err4", "404"))
	if after-before != 1 {
		t.Fatalf("expected error counter to increment by 1, got %f", after-before)
	}
}

func TestMetrics_ErrorCounterIncrements5xx(t *testing.T) {
	before := testutil.ToFloat64(httpRequestErrorsTotal.WithLabelValues("POST", "/err5", "500"))

	handler := Metrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodPost, "/err5", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	after := testutil.ToFloat64(httpRequestErrorsTotal.WithLabelValues("POST", "/err5", "500"))
	if after-before != 1 {
		t.Fatalf("expected error counter to increment by 1, got %f", after-before)
	}
}

func TestMetrics_CallsNextHandler(t *testing.T) {
	called := false
	handler := Metrics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/next", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("next handler was not called")
	}
}

func TestMetrics_EndpointResponds(t *testing.T) {
	// Verify the promhttp handler serves metrics successfully
	handler := promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{})
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("metrics endpoint returned %d, want 200", rr.Code)
	}
	if rr.Header().Get("Content-Type") == "" {
		t.Fatal("metrics endpoint returned no content-type")
	}
}
