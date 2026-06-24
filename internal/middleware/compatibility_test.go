package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
)

// TestCompatibility_LegacyAuthOnly proves the legacy path still works
// when only AuthenticatedUser is present (no MembershipContext).
func TestCompatibility_LegacyAuthOnly(t *testing.T) {
	var gotOrgID string
	handler := TenantContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := appcontext.GetOrganizationID(r.Context())
		gotOrgID = orgID
	}))

	req := httptest.NewRequest("GET", "/", nil)
	ctx := appcontext.SetAuthenticatedUser(req.Context(), &appcontext.AuthenticatedUser{
		UserID: "user-1", OrganizationID: "legacy-org",
	})
	req = req.WithContext(ctx)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotOrgID != "legacy-org" {
		t.Fatalf("expected legacy-org, got %s", gotOrgID)
	}
}

// TestCompatibility_MembershipOnly proves the new path works
// when only MembershipContext is present.
func TestCompatibility_MembershipOnly(t *testing.T) {
	var gotOrgID string
	handler := TenantContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := appcontext.GetOrganizationID(r.Context())
		gotOrgID = orgID
	}))

	req := httptest.NewRequest("GET", "/", nil)
	ctx := appcontext.SetMembership(req.Context(), &appcontext.Membership{
		MembershipID: "mem-1", OrganizationID: "membership-org",
	})
	req = req.WithContext(ctx)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotOrgID != "membership-org" {
		t.Fatalf("expected membership-org, got %s", gotOrgID)
	}
}

// TestCompatibility_BothPresent_MembershipWins proves MembershipContext
// takes precedence over AuthenticatedUser when both are set.
func TestCompatibility_BothPresent_MembershipWins(t *testing.T) {
	var gotOrgID string
	handler := TenantContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID, _ := appcontext.GetOrganizationID(r.Context())
		gotOrgID = orgID
	}))

	req := httptest.NewRequest("GET", "/", nil)
	ctx := appcontext.SetAuthenticatedUser(req.Context(), &appcontext.AuthenticatedUser{
		UserID: "user-1", OrganizationID: "old-org",
	})
	ctx = appcontext.SetMembership(ctx, &appcontext.Membership{
		MembershipID: "mem-1", OrganizationID: "new-org",
	})
	req = req.WithContext(ctx)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotOrgID != "new-org" {
		t.Fatalf("expected new-org (membership wins), got %s", gotOrgID)
	}
}

// TestMembershipResolution_FullChain proves the middleware chain:
// IdentityContext → ResolveMembership → MembershipContext → TenantContext
func TestMembershipResolution_FullChain(t *testing.T) {
	var gotOrgID string

	resolver := &mockMembershipResolver{
		membership: &appcontext.Membership{MembershipID: "mem-99", OrganizationID: "resolved-org"},
	}

	// Chain: ResolveMembership → TenantContext → handler
	chain := ResolveMembershipMiddleware(resolver)(
		TenantContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID, _ := appcontext.GetOrganizationID(r.Context())
			gotOrgID = orgID
		})),
	)

	req := httptest.NewRequest("GET", "/", nil)
	ctx := appcontext.SetIdentity(req.Context(), &appcontext.Identity{
		IdentityID: "id-1", ProviderID: "prov-1", Email: "a@b.com",
	})
	req = req.WithContext(ctx)
	chain.ServeHTTP(httptest.NewRecorder(), req)

	if gotOrgID != "resolved-org" {
		t.Fatalf("expected resolved-org from full chain, got %s", gotOrgID)
	}
}
