package identity

import (
	"encoding/json"
	"testing"
)

func TestClaimsUnmarshalCapturesTypedAndRawClaims(t *testing.T) {
	payload := []byte(`{
		"iss": "http://issuer.example",
		"sub": "user-123",
		"aud": ["client-123"],
		"exp": 4102444800,
		"email": "test@example.com",
		"email_verified": true,
		"preferred_username": "tester",
		"name": "Test User",
		"given_name": "Test",
		"family_name": "User",
		"locale": "en",
		"nonce": "nonce-123",
		"organization_id": "org-123",
		"custom_claim": {"enabled": true},
		"roles": ["admin", "editor"]
	}`)

	var claims Claims

	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("unmarshal claims: %v", err)
	}

	if claims.Subject != "user-123" {
		t.Fatalf("expected subject, got %q", claims.Subject)
	}

	if claims.UserID != "user-123" {
		t.Fatalf("expected user id fallback from subject, got %q", claims.UserID)
	}

	if claims.Email != "test@example.com" {
		t.Fatalf("expected email, got %q", claims.Email)
	}

	if !claims.EmailVerified {
		t.Fatal("expected verified email")
	}

	if claims.PreferredUsername != "tester" {
		t.Fatalf("expected preferred username, got %q", claims.PreferredUsername)
	}

	if claims.Name != "Test User" {
		t.Fatalf("expected name, got %q", claims.Name)
	}

	if claims.OrganizationID != "org-123" {
		t.Fatalf("expected organization id, got %q", claims.OrganizationID)
	}

	if _, ok := claims.RawClaims["custom_claim"].(map[string]any); !ok {
		t.Fatalf("expected custom claim in raw claims, got %#v", claims.RawClaims["custom_claim"])
	}

	if _, ok := claims.RawClaims["roles"].([]any); !ok {
		t.Fatalf("expected roles array in raw claims, got %#v", claims.RawClaims["roles"])
	}
}
