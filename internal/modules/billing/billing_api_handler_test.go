package billing

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/internal/modules/entitlements"
)

type fakeUsageStore struct {
	values map[string]int64
	err    error
}

func (f *fakeUsageStore) GetUsage(ctx context.Context, organizationID string, metric string) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.values[metric], nil
}

type fakeSubGetter struct {
	plan   string
	status string
}

func (f *fakeSubGetter) GetSubscription(ctx context.Context, organizationID string) (string, string, error) {
	return f.plan, f.status, nil
}

func apiRequest(orgID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := appcontext.SetOrganizationID(req.Context(), orgID)
	return req.WithContext(ctx)
}

func TestGetSubscriptionActive(t *testing.T) {
	store := &fakeBillingStore{
		getSubscription: &Subscription{
			Plan:     "pro",
			Status:   "active",
			Provider: "dodo",
			CurrentPeriodEnd: func() *time.Time {
				v := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
				return &v
			}(),
		},
	}
	svc := NewService(store, nil, nil)
	h := NewBillingAPIHandler(svc, &fakeUsageStore{}, entitlements.NewService(&fakeSubGetter{plan: "pro", status: "active"}, &fakeUsageStore{}))

	rec := httptest.NewRecorder()
	h.GetSubscription(rec, apiRequest("org-1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"plan":"pro"`) {
		t.Fatalf("expected plan in response, got %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"active"`) {
		t.Fatalf("expected status in response, got %s", rec.Body.String())
	}
}

func TestGetSubscriptionMissing(t *testing.T) {
	store := &fakeBillingStore{getSubscription: nil}
	svc := NewService(store, nil, nil)
	h := NewBillingAPIHandler(svc, &fakeUsageStore{}, entitlements.NewService(&fakeSubGetter{}, &fakeUsageStore{}))

	rec := httptest.NewRecorder()
	h.GetSubscription(rec, apiRequest("org-1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "null") {
		t.Fatalf("expected null data, got %s", rec.Body.String())
	}
}

func TestGetUsageReturnsValues(t *testing.T) {
	usage := &fakeUsageStore{values: map[string]int64{
		"users": 5, "documents": 100, "api_requests": 500, "storage": 1024,
	}}
	h := NewBillingAPIHandler(nil, usage, nil)

	rec := httptest.NewRecorder()
	h.GetUsage(rec, apiRequest("org-1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"users":5`) {
		t.Fatalf("expected users in response, got %s", body)
	}
	if !strings.Contains(body, `"documents":100`) {
		t.Fatalf("expected documents in response, got %s", body)
	}
}

func TestGetEntitlementsReturnsValues(t *testing.T) {
	usage := &fakeUsageStore{values: map[string]int64{"users": 1}}
	entSvc := entitlements.NewService(
		&fakeSubGetter{plan: "free", status: "active"},
		usage,
	)
	h := NewBillingAPIHandler(nil, usage, entSvc)

	rec := httptest.NewRecorder()
	h.GetEntitlements(rec, apiRequest("org-1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"metric":"users"`) {
		t.Fatalf("expected users metric in response, got %s", body)
	}
	if !strings.Contains(body, `"entitlements"`) {
		t.Fatalf("expected entitlements key, got %s", body)
	}
}

func TestGetUsageServiceFailure(t *testing.T) {
	usage := &fakeUsageStore{err: errors.New("db error")}
	h := NewBillingAPIHandler(nil, usage, nil)

	rec := httptest.NewRecorder()
	h.GetUsage(rec, apiRequest("org-1"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestGetSubscriptionMissingOrgContext(t *testing.T) {
	h := NewBillingAPIHandler(NewService(&fakeBillingStore{}, nil, nil), &fakeUsageStore{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.GetSubscription(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestGetUsageMissingOrgContext(t *testing.T) {
	h := NewBillingAPIHandler(nil, &fakeUsageStore{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.GetUsage(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestGetEntitlementsMissingOrgContext(t *testing.T) {
	h := NewBillingAPIHandler(nil, &fakeUsageStore{}, entitlements.NewService(&fakeSubGetter{}, &fakeUsageStore{}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.GetEntitlements(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
