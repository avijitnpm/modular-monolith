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
	"github.com/go-chi/chi/v5"
)

func TestHandlerCreateBillingCreatesSubscription(t *testing.T) {
	store := &fakeBillingStore{
		createSubscription: testSubscription(),
	}
	handler := NewHandler(NewService(store, nil, nil))
	req := billingRequest(
		http.MethodPost,
		"/api/v1/billing",
		`{"provider":"manual","plan":"free","status":"active"}`,
		"org-1",
	)
	recorder := httptest.NewRecorder()

	handler.CreateBilling(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", recorder.Code)
	}

	if store.organizationID != "org-1" {
		t.Fatalf("expected organization org-1, got %q", store.organizationID)
	}

	if !strings.Contains(recorder.Body.String(), `"provider":"manual"`) {
		t.Fatalf("expected subscription response, got %s", recorder.Body.String())
	}
}

func TestHandlerCreateCheckoutReturnsURL(t *testing.T) {
	handler := NewHandler(
		NewService(
			&fakeBillingStore{},
			&fakeCheckoutProvider{
				url: "https://checkout.example/session-1",
			},
			nil,
		),
	)
	req := billingRequest(
		http.MethodPost,
		"/api/v1/billing/checkout",
		`{"plan":"prod_basic"}`,
		"org-1",
	)
	recorder := httptest.NewRecorder()

	handler.CreateCheckout(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), `"checkout_url":"https://checkout.example/session-1"`) {
		t.Fatalf("expected checkout url response, got %s", recorder.Body.String())
	}
}

func TestHandlerCreateCheckoutRejectsMissingPlan(t *testing.T) {
	handler := NewHandler(
		NewService(
			&fakeBillingStore{},
			&fakeCheckoutProvider{},
			nil,
		),
	)
	req := billingRequest(
		http.MethodPost,
		"/api/v1/billing/checkout",
		`{"plan":" "}`,
		"org-1",
	)
	recorder := httptest.NewRecorder()

	handler.CreateCheckout(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestHandlerCreateBillingRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewService(&fakeBillingStore{}, nil, nil))
	req := billingRequest(
		http.MethodPost,
		"/api/v1/billing",
		`{`,
		"org-1",
	)
	recorder := httptest.NewRecorder()

	handler.CreateBilling(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestHandlerCreateBillingRejectsMissingFields(t *testing.T) {
	handler := NewHandler(NewService(&fakeBillingStore{}, nil, nil))
	req := billingRequest(
		http.MethodPost,
		"/api/v1/billing",
		`{"provider":"manual","plan":"","status":"active"}`,
		"org-1",
	)
	recorder := httptest.NewRecorder()

	handler.CreateBilling(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestHandlerCreateBillingReturnsConflict(t *testing.T) {
	handler := NewHandler(
		NewService(
			&fakeBillingStore{
				createErr: ErrSubscriptionAlreadyExists,
			},
			nil,
			nil,
		),
	)
	req := billingRequest(
		http.MethodPost,
		"/api/v1/billing",
		`{"provider":"manual","plan":"free","status":"active"}`,
		"org-1",
	)
	recorder := httptest.NewRecorder()

	handler.CreateBilling(recorder, req)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", recorder.Code)
	}
}

func TestHandlerUpdateBillingUpdatesSubscription(t *testing.T) {
	currentPeriodEnd := time.Now().UTC().Add(30 * 24 * time.Hour)
	store := &fakeBillingStore{
		updateSubscription: &Subscription{
			ID:               "subscription-1",
			OrganizationID:   "org-1",
			Provider:         "manual",
			Plan:             "pro",
			Status:           "active",
			CurrentPeriodEnd: &currentPeriodEnd,
		},
	}
	handler := NewHandler(NewService(store, nil, nil))
	req := billingUpdateRequest(
		http.MethodPatch,
		"/api/v1/billing/subscription-1",
		`{"plan":"pro","status":"active","current_period_end":"`+
			currentPeriodEnd.Format(time.RFC3339Nano)+`"}`,
		"org-1",
		"subscription-1",
	)
	recorder := httptest.NewRecorder()

	handler.UpdateBilling(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	if store.id != "subscription-1" ||
		store.plan != "pro" ||
		store.status != "active" ||
		store.currentPeriodEnd == nil ||
		!store.currentPeriodEnd.Equal(currentPeriodEnd) {
		t.Fatalf(
			"expected update fields, got id=%q plan=%q status=%q current_period_end=%v",
			store.id,
			store.plan,
			store.status,
			store.currentPeriodEnd,
		)
	}
}

func TestHandlerUpdateBillingReturnsNotFound(t *testing.T) {
	handler := NewHandler(
		NewService(
			&fakeBillingStore{
				updateErr: ErrSubscriptionNotFound,
			},
			nil,
			nil,
		),
	)
	req := billingUpdateRequest(
		http.MethodPatch,
		"/api/v1/billing/missing-subscription",
		`{"plan":"pro","status":"active"}`,
		"org-1",
		"missing-subscription",
	)
	recorder := httptest.NewRecorder()

	handler.UpdateBilling(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Code)
	}
}

func TestHandlerUpdateBillingReturnsInternalError(t *testing.T) {
	handler := NewHandler(
		NewService(
			&fakeBillingStore{
				updateErr: errors.New("update failed"),
			},
			nil,
			nil,
		),
	)
	req := billingUpdateRequest(
		http.MethodPatch,
		"/api/v1/billing/subscription-1",
		`{"plan":"pro","status":"active"}`,
		"org-1",
		"subscription-1",
	)
	recorder := httptest.NewRecorder()

	handler.UpdateBilling(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", recorder.Code)
	}
}

func TestHandlerGetBillingRejectsMissingOrganizationContext(t *testing.T) {
	handler := NewHandler(NewService(&fakeBillingStore{}, nil, nil))
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/billing",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.GetBilling(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", recorder.Code)
	}
}

func billingRequest(
	method string,
	target string,
	body string,
	organizationID string,
) *http.Request {

	req := httptest.NewRequest(
		method,
		target,
		strings.NewReader(body),
	)

	ctx := appcontext.SetOrganizationID(
		req.Context(),
		organizationID,
	)

	return req.WithContext(ctx)
}

func billingUpdateRequest(
	method string,
	target string,
	body string,
	organizationID string,
	subscriptionID string,
) *http.Request {

	req := billingRequest(
		method,
		target,
		body,
		organizationID,
	)

	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(
		"id",
		subscriptionID,
	)

	ctx := context.WithValue(
		req.Context(),
		chi.RouteCtxKey,
		routeContext,
	)

	return req.WithContext(ctx)
}
