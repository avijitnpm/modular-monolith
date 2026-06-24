package billing

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
)

type mockStore struct {
	subscription *Subscription
	err          error
}

func (m *mockStore) GetSubscription(ctx context.Context, organizationID string) (*Subscription, error) {
	return m.subscription, m.err
}

func (m *mockStore) CreateSubscription(ctx context.Context, organizationID, provider, plan, status string) (*Subscription, error) {
	return nil, nil
}

func (m *mockStore) UpdateSubscription(ctx context.Context, id, organizationID, plan, status string, currentPeriodEnd *time.Time) (*Subscription, error) {
	return nil, nil
}

func (m *mockStore) UpsertSubscriptionByProvider(ctx context.Context, organizationID, provider, providerSubscriptionID, providerCustomerID, plan, status string, currentPeriodEnd *time.Time) error {
	return nil
}

func TestHasActiveSubscription(t *testing.T) {
	tests := []struct {
		name   string
		sub    *Subscription
		err    error
		want   bool
		wantErr bool
	}{
		{"active", &Subscription{Status: "active"}, nil, true, false},
		{"trialing", &Subscription{Status: "trialing"}, nil, true, false},
		{"cancelled", &Subscription{Status: "cancelled"}, nil, false, false},
		{"past_due", &Subscription{Status: "past_due"}, nil, false, false},
		{"nil subscription", nil, nil, false, false},
		{"store error", nil, errors.New("db error"), false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewSubscriptionChecker(&mockStore{subscription: tt.sub, err: tt.err})
			got, err := checker.HasActiveSubscription(context.Background(), "org-1")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPlan(t *testing.T) {
	checker := NewSubscriptionChecker(&mockStore{subscription: &Subscription{Plan: "pro"}})
	got, _ := checker.IsPlan(context.Background(), "org-1", "pro")
	if !got {
		t.Fatal("expected true for matching plan")
	}
	got, _ = checker.IsPlan(context.Background(), "org-1", "enterprise")
	if got {
		t.Fatal("expected false for non-matching plan")
	}
}

func TestRequireSubscriptionMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		sub        *Subscription
		err        error
		wantStatus int
	}{
		{"active passes", &Subscription{Status: "active"}, nil, http.StatusOK},
		{"trialing passes", &Subscription{Status: "trialing"}, nil, http.StatusOK},
		{"cancelled blocks", &Subscription{Status: "cancelled"}, nil, http.StatusForbidden},
		{"past_due blocks", &Subscription{Status: "past_due"}, nil, http.StatusForbidden},
		{"missing blocks", nil, nil, http.StatusForbidden},
		{"error blocks", nil, errors.New("db error"), http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewSubscriptionChecker(&mockStore{subscription: tt.sub, err: tt.err})
			handler := RequireSubscription(checker)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			ctx := appcontext.SetOrganizationID(req.Context(), "org-1")
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("got status %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestRequireSubscriptionMiddleware_NoOrgContext(t *testing.T) {
	checker := NewSubscriptionChecker(&mockStore{subscription: &Subscription{Status: "active"}})
	handler := RequireSubscription(checker)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("got status %d, want %d", rec.Code, http.StatusForbidden)
	}
}
