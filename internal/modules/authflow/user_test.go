package authflow

import (
	"reflect"
	"testing"

	"github.com/avijitnpm/modular-monolith/internal/providers/identity"
)

func TestNormalizeUserMapsTypedAndRawClaims(t *testing.T) {
	claims := &identity.Claims{
		UserID:            "user-123",
		Email:             "test@example.com",
		EmailVerified:     true,
		PreferredUsername: "tester",
		Name:              "Test User",
		GivenName:         "Test",
		FamilyName:        "User",
		Locale:            "en",
		RawClaims: map[string]any{
			"sub":                                "user-123",
			"urn:zitadel:iam:org:id":             "org-raw",
			"urn:zitadel:iam:org:project:roles":  map[string]any{"admin": map[string]any{"org": "org-raw"}},
			"https://example.com/custom_roles":   []any{"editor", "viewer"},
			"https://example.com/feature_toggle": true,
		},
	}

	user := normalizeUser(claims)

	if user.Subject != "user-123" {
		t.Fatalf("expected subject, got %q", user.Subject)
	}

	if user.Email != "test@example.com" {
		t.Fatalf("expected email, got %q", user.Email)
	}

	if !user.EmailVerified {
		t.Fatal("expected verified email")
	}

	if user.Name != "Test User" {
		t.Fatalf("expected name, got %q", user.Name)
	}

	if user.OrganizationID != "org-raw" {
		t.Fatalf("expected organization id from raw claims, got %q", user.OrganizationID)
	}

	expectedRoles := []string{
		"admin",
		"editor",
		"viewer",
	}

	if !reflect.DeepEqual(user.Roles, expectedRoles) {
		t.Fatalf("expected roles %#v, got %#v", expectedRoles, user.Roles)
	}

	if user.RawClaims["sub"] != "user-123" {
		t.Fatalf("expected raw claims to be preserved, got %#v", user.RawClaims)
	}
}

func TestNormalizeUserPrefersTypedOrganizationID(t *testing.T) {
	user := normalizeUser(
		&identity.Claims{
			UserID:         "user-123",
			OrganizationID: "org-typed",
			RawClaims: map[string]any{
				"urn:zitadel:iam:org:id": "org-raw",
			},
		},
	)

	if user.OrganizationID != "org-typed" {
		t.Fatalf("expected typed organization id, got %q", user.OrganizationID)
	}
}
