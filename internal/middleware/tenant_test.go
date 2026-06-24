package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
)

func TestTenantContext_MembershipPreferred(t *testing.T) {
	var gotOrgID string

	handler := TenantContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := appcontext.GetOrganizationID(r.Context())
		gotOrgID = orgID
	}))

	req := httptest.NewRequest("GET", "/", nil)
	ctx := appcontext.SetAuthenticatedUser(req.Context(), &appcontext.AuthenticatedUser{
		UserID: "u1", OrganizationID: "old-org",
	})
	ctx = appcontext.SetMembership(ctx, &appcontext.Membership{
		MembershipID: "m1", OrganizationID: "new-org",
	})
	req = req.WithContext(ctx)

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotOrgID != "new-org" {
		t.Fatalf("expected new-org, got %s", gotOrgID)
	}
}

func TestTenantContext_FallbackToAuthenticatedUser(t *testing.T) {
	var gotOrgID string

	handler := TenantContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := appcontext.GetOrganizationID(r.Context())
		gotOrgID = orgID
	}))

	req := httptest.NewRequest("GET", "/", nil)
	ctx := appcontext.SetAuthenticatedUser(req.Context(), &appcontext.AuthenticatedUser{
		UserID: "u1", OrganizationID: "fallback-org",
	})
	req = req.WithContext(ctx)

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotOrgID != "fallback-org" {
		t.Fatalf("expected fallback-org, got %s", gotOrgID)
	}
}

func TestTenantContext_NoContext_Returns500(t *testing.T) {
	handler := TenantContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach handler")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
