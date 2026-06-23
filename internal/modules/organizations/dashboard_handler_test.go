package organizations

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/internal/modules/entitlements"
)

type mockOrgGetter struct {
	name string
	err  error
}

func (m *mockOrgGetter) GetOrganizationName(ctx context.Context, orgID string) (string, error) {
	return m.name, m.err
}

type mockSubGetter struct {
	plan   string
	status string
}

func (m *mockSubGetter) GetSubscription(ctx context.Context, orgID string) (string, string, error) {
	return m.plan, m.status, nil
}

type mockUsageGetter struct {
	values map[string]int64
}

func (m *mockUsageGetter) GetUsage(ctx context.Context, orgID string, metric string) (int64, error) {
	return m.values[metric], nil
}

func dashboardRequest(orgID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := appcontext.SetOrganizationID(req.Context(), orgID)
	return req.WithContext(ctx)
}

func testHandler() *DashboardHandler {
	return NewDashboardHandler(
		&mockOrgGetter{name: "Acme Inc"},
		&mockSubGetter{plan: "pro", status: "active"},
		&mockUsageGetter{values: map[string]int64{"users": 3, "documents": 100, "api_requests": 500, "storage": 2048}},
		entitlements.NewService(
			&mockSubGetter{plan: "pro", status: "active"},
			&mockUsageGetter{values: map[string]int64{"users": 3, "documents": 100, "api_requests": 500, "storage": 2048}},
		),
	)
}

func TestDashboardSuccess(t *testing.T) {
	h := testHandler()
	rec := httptest.NewRecorder()
	h.Dashboard(rec, dashboardRequest("org-1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"name":"Acme Inc"`) {
		t.Fatalf("expected org name, got %s", body)
	}
	if !strings.Contains(body, `"plan":"pro"`) {
		t.Fatalf("expected plan, got %s", body)
	}
	if !strings.Contains(body, `"users":3`) {
		t.Fatalf("expected usage, got %s", body)
	}
	if !strings.Contains(body, `"entitlements"`) {
		t.Fatalf("expected entitlements, got %s", body)
	}
}

func TestSummarySuccess(t *testing.T) {
	h := testHandler()
	rec := httptest.NewRecorder()
	h.Summary(rec, dashboardRequest("org-1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"organization_name":"Acme Inc"`) {
		t.Fatalf("expected org name, got %s", body)
	}
	if !strings.Contains(body, `"plan":"pro"`) {
		t.Fatalf("expected plan, got %s", body)
	}
}

func TestUsageSummarySuccess(t *testing.T) {
	h := testHandler()
	rec := httptest.NewRecorder()
	h.UsageSummary(rec, dashboardRequest("org-1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"users":3`) {
		t.Fatalf("expected users, got %s", body)
	}
	if !strings.Contains(body, `"storage":2048`) {
		t.Fatalf("expected storage, got %s", body)
	}
}

func TestDashboardMissingOrgContext(t *testing.T) {
	h := testHandler()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.Dashboard(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestSummaryMissingOrgContext(t *testing.T) {
	h := testHandler()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.Summary(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestDashboardOrgGetterFailure(t *testing.T) {
	h := NewDashboardHandler(
		&mockOrgGetter{err: errors.New("db error")},
		&mockSubGetter{},
		&mockUsageGetter{values: map[string]int64{}},
		entitlements.NewService(&mockSubGetter{}, &mockUsageGetter{values: map[string]int64{}}),
	)
	rec := httptest.NewRecorder()
	h.Dashboard(rec, dashboardRequest("org-1"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
