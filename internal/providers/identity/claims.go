package identity

import (
	"encoding/json"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"user_id"`

	OrganizationID string `json:"organization_id"`

	Email string `json:"email"`

	EmailVerified bool `json:"email_verified"`

	PreferredUsername string `json:"preferred_username"`

	Name string `json:"name"`

	GivenName string `json:"given_name"`

	FamilyName string `json:"family_name"`

	Locale string `json:"locale"`

	Nonce string `json:"nonce"`

	RawClaims map[string]any `json:"-"`

	jwt.RegisteredClaims
}

func (c *Claims) UnmarshalJSON(data []byte) error {
	type claimsJSON struct {
		UserID            string `json:"user_id"`
		OrganizationID    string `json:"organization_id"`
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		PreferredUsername string `json:"preferred_username"`
		Name              string `json:"name"`
		GivenName         string `json:"given_name"`
		FamilyName        string `json:"family_name"`
		Locale            string `json:"locale"`
		Nonce             string `json:"nonce"`
		jwt.RegisteredClaims
	}

	var raw map[string]any

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var parsed claimsJSON

	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}

	c.UserID = parsed.UserID
	c.OrganizationID = parsed.OrganizationID
	c.Email = parsed.Email
	c.EmailVerified = parsed.EmailVerified
	c.PreferredUsername = parsed.PreferredUsername
	c.Name = parsed.Name
	c.GivenName = parsed.GivenName
	c.FamilyName = parsed.FamilyName
	c.Locale = parsed.Locale
	c.Nonce = parsed.Nonce
	c.RegisteredClaims = parsed.RegisteredClaims
	c.RawClaims = raw

	if c.UserID == "" {
		c.UserID = c.Subject
	}

	return nil
}
