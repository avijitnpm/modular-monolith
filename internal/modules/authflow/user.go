package authflow

import (
	"sort"
	"strings"

	"github.com/avijitnpm/modular-monolith/internal/providers/identity"
)

type SessionUser struct {
	Subject           string `json:"subject"`
	IdentityID        string `json:"identity_id,omitempty"`
	Email             string `json:"email,omitempty"`
	EmailVerified     bool   `json:"email_verified,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
	Name              string `json:"name,omitempty"`
	GivenName         string `json:"given_name,omitempty"`
	FamilyName        string `json:"family_name,omitempty"`
	Locale            string `json:"locale,omitempty"`
	// OrganizationID is legacy. Populated from OIDC claims but no longer
	// drives routing or authorization. New code should not depend on this field.
	OrganizationID string         `json:"organization_id,omitempty"`
	Roles          []string       `json:"roles,omitempty"`
	RawClaims      map[string]any `json:"raw_claims,omitempty"`
}

func normalizeUser(
	claims *identity.Claims,
) SessionUser {
	user := SessionUser{
		Subject:           claims.UserID,
		Email:             claims.Email,
		EmailVerified:     claims.EmailVerified,
		PreferredUsername: claims.PreferredUsername,
		Name:              claims.Name,
		GivenName:         claims.GivenName,
		FamilyName:        claims.FamilyName,
		Locale:            claims.Locale,
		OrganizationID:    claims.OrganizationID,
		Roles:             extractRoles(claims.RawClaims),
		RawClaims:         claims.RawClaims,
	}

	if user.Subject == "" {
		user.Subject = claims.Subject
	}

	if user.OrganizationID == "" {
		user.OrganizationID = extractOrganizationID(claims.RawClaims)
	}

	return user
}

func extractOrganizationID(
	claims map[string]any,
) string {
	for _, key := range []string{
		"organization_id",
		"org_id",
		"urn:zitadel:iam:org:id",
	} {
		if value, ok := stringClaim(claims[key]); ok {
			return value
		}
	}

	for key, value := range claims {
		lowerKey := strings.ToLower(key)

		if !strings.Contains(lowerKey, "org") &&
			!strings.Contains(lowerKey, "organization") {
			continue
		}

		if strings.Contains(lowerKey, "role") {
			continue
		}

		if value, ok := stringClaim(value); ok {
			return value
		}
	}

	return ""
}

func extractRoles(
	claims map[string]any,
) []string {
	roleSet := map[string]struct{}{}

	for key, value := range claims {
		if !strings.Contains(
			strings.ToLower(key),
			"role",
		) {
			continue
		}

		addRoles(
			roleSet,
			value,
		)
	}

	roles := make(
		[]string,
		0,
		len(roleSet),
	)

	for role := range roleSet {
		roles = append(
			roles,
			role,
		)
	}

	sort.Strings(roles)

	return roles
}

func addRoles(
	roles map[string]struct{},
	value any,
) {
	switch typed := value.(type) {
	case string:
		if typed != "" {
			roles[typed] = struct{}{}
		}
	case []string:
		for _, role := range typed {
			if role != "" {
				roles[role] = struct{}{}
			}
		}
	case []any:
		for _, item := range typed {
			if role, ok := stringClaim(item); ok {
				roles[role] = struct{}{}
			}
		}
	case map[string]bool:
		for role, enabled := range typed {
			if enabled && role != "" {
				roles[role] = struct{}{}
			}
		}
	case map[string]any:
		for role, enabled := range typed {
			if isEnabledRole(enabled) && role != "" {
				roles[role] = struct{}{}
			}
		}
	}
}

func isEnabledRole(
	value any,
) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed != "" && typed != "false"
	case map[string]any:
		return len(typed) > 0
	default:
		return value != nil
	}
}

func stringClaim(
	value any,
) (string, bool) {
	stringValue, ok := value.(string)

	if !ok || stringValue == "" {
		return "", false
	}

	return stringValue, true
}
