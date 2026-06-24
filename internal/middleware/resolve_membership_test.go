package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
)

type mockMembershipResolver struct {
	membership *appcontext.Membership
	err        error
}

func (m *mockMembershipResolver) ResolveMembership(_ context.Context, identityID string) (*appcontext.Membership, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.membership, nil
}

func TestResolveMembershipMiddleware_SetsContext(t *testing.T) {
	var gotMembership *appcontext.Membership

	resolver := &mockMembershipResolver{
		membership: &appcontext.Membership{MembershipID: "mem-1", OrganizationID: "org-1"},
	}

	handler := ResolveMembershipMiddleware(resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m, ok := appcontext.GetMembership(r.Context())
		if ok {
			gotMembership = m
		}
	}))

	req := httptest.NewRequest("GET", "/", nil)
	ctx := appcontext.SetIdentity(req.Context(), &appcontext.Identity{IdentityID: "id-1"})
	req = req.WithContext(ctx)

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotMembership == nil || gotMembership.MembershipID != "mem-1" {
		t.Fatalf("expected membership mem-1, got %+v", gotMembership)
	}
}

func TestResolveMembershipMiddleware_NoIdentity_Proceeds(t *testing.T) {
	called := false
	resolver := &mockMembershipResolver{
		membership: &appcontext.Membership{MembershipID: "mem-1", OrganizationID: "org-1"},
	}

	handler := ResolveMembershipMiddleware(resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, ok := appcontext.GetMembership(r.Context())
		if ok {
			t.Fatal("expected no membership when no identity")
		}
	}))

	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("handler was not called")
	}
}
