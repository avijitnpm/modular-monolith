package context

import (
	"context"
	"testing"
)

func TestIdentityContext(t *testing.T) {
	ctx := context.Background()

	_, ok := GetIdentity(ctx)
	if ok {
		t.Fatal("expected no identity in empty context")
	}

	id := &Identity{IdentityID: "id-1", ProviderID: "prov-1", Email: "a@b.com", Name: "Alice"}
	ctx = SetIdentity(ctx, id)

	got, ok := GetIdentity(ctx)
	if !ok {
		t.Fatal("expected identity in context")
	}
	if got.IdentityID != "id-1" || got.ProviderID != "prov-1" || got.Email != "a@b.com" || got.Name != "Alice" {
		t.Fatalf("unexpected identity: %+v", got)
	}
}

func TestMembershipContext(t *testing.T) {
	ctx := context.Background()

	_, ok := GetMembership(ctx)
	if ok {
		t.Fatal("expected no membership in empty context")
	}

	m := &Membership{MembershipID: "mem-1", OrganizationID: "org-1"}
	ctx = SetMembership(ctx, m)

	got, ok := GetMembership(ctx)
	if !ok {
		t.Fatal("expected membership in context")
	}
	if got.MembershipID != "mem-1" || got.OrganizationID != "org-1" {
		t.Fatalf("unexpected membership: %+v", got)
	}
}

func TestGetCurrentOrganizationID_MembershipFirst(t *testing.T) {
	ctx := context.Background()
	ctx = SetAuthenticatedUser(ctx, &AuthenticatedUser{OrganizationID: "old-org"})
	ctx = SetMembership(ctx, &Membership{OrganizationID: "new-org"})

	if got := GetCurrentOrganizationID(ctx); got != "new-org" {
		t.Fatalf("expected new-org, got %s", got)
	}
}

func TestGetCurrentOrganizationID_FallbackToUser(t *testing.T) {
	ctx := context.Background()
	ctx = SetAuthenticatedUser(ctx, &AuthenticatedUser{OrganizationID: "user-org"})

	if got := GetCurrentOrganizationID(ctx); got != "user-org" {
		t.Fatalf("expected user-org, got %s", got)
	}
}

func TestGetCurrentOrganizationID_Empty(t *testing.T) {
	ctx := context.Background()

	if got := GetCurrentOrganizationID(ctx); got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}
