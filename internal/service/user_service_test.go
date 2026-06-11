package service

import "testing"

func TestProvisionedUserEmailUsesProvidedEmail(t *testing.T) {
	email := provisionedUserEmail(
		"371126430606098435",
		" user@example.com ",
	)

	if email != "user@example.com" {
		t.Fatalf("expected trimmed email, got %q", email)
	}
}

func TestProvisionedUserEmailFallsBackToSyntheticEmail(t *testing.T) {
	email := provisionedUserEmail(
		"371126430606098435",
		"",
	)

	if email != "371126430606098435@zitadel.local" {
		t.Fatalf("expected synthetic email, got %q", email)
	}
}
