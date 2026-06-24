package authflow

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/providers/identity"
	"github.com/golang-jwt/jwt/v5"
)

func TestLoginSetsPKCECookiesAndRedirects(t *testing.T) {
	oauth := &mockOAuthProvider{
		authorizationURL: "http://issuer.example/authorize",
	}
	handler := newTestHandler(t, oauth, &mockTokenValidator{})

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/login",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.Login(
		rec,
		req,
	)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}

	if rec.Header().Get("Location") != oauth.authorizationURL {
		t.Fatalf("expected redirect to auth url, got %q", rec.Header().Get("Location"))
	}

	assertCookie(t, rec.Result(), stateCookieName)
	assertCookie(t, rec.Result(), nonceCookieName)
	assertCookie(t, rec.Result(), codeVerifierCookieName)

	if oauth.request.State == "" {
		t.Fatal("expected state in authorization request")
	}

	if oauth.request.Nonce == "" {
		t.Fatal("expected nonce in authorization request")
	}

	if oauth.request.CodeChallenge == "" {
		t.Fatal("expected code challenge in authorization request")
	}

	if oauth.request.Scope != "openid email profile" {
		t.Fatalf("expected default scope, got %q", oauth.request.Scope)
	}
}

func TestCallbackRejectsInvalidState(t *testing.T) {
	handler := newTestHandler(
		t,
		&mockOAuthProvider{},
		&mockTokenValidator{},
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/callback?code=abc&state=returned",
		nil,
	)
	req.AddCookie(testCookie(stateCookieName, "stored"))
	req.AddCookie(testCookie(nonceCookieName, "nonce"))
	req.AddCookie(testCookie(codeVerifierCookieName, "verifier"))

	rec := httptest.NewRecorder()

	handler.Callback(
		rec,
		req,
	)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", rec.Code)
	}
}

func TestCallbackCreatesSessionAndMeReturnsUser(t *testing.T) {
	oauth := &mockOAuthProvider{
		token: &identity.TokenResponse{
			IDToken: "id-token",
		},
	}
	validator := &mockTokenValidator{
		claims: &identity.Claims{
			UserID:            "user-123",
			Email:             "test@example.com",
			EmailVerified:     true,
			PreferredUsername: "tester",
			Name:              "Test User",
			GivenName:         "Test",
			FamilyName:        "User",
			Nonce:             "nonce",
			RawClaims: map[string]any{
				"sub":                               "user-123",
				"email":                             "test@example.com",
				"email_verified":                    true,
				"preferred_username":                "tester",
				"name":                              "Test User",
				"given_name":                        "Test",
				"family_name":                       "User",
				"urn:zitadel:iam:org:id":            "org-123",
				"urn:zitadel:iam:org:project:roles": map[string]any{"admin": true},
			},
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(
					time.Now().Add(time.Hour),
				),
			},
		},
	}
	handler := newTestHandler(
		t,
		oauth,
		validator,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/callback?code=abc&state=stored",
		nil,
	)
	req.AddCookie(testCookie(stateCookieName, "stored"))
	req.AddCookie(testCookie(nonceCookieName, "nonce"))
	req.AddCookie(testCookie(codeVerifierCookieName, "verifier"))

	rec := httptest.NewRecorder()

	handler.Callback(
		rec,
		req,
	)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}

	if oauth.code != "abc" {
		t.Fatalf("expected exchanged code, got %q", oauth.code)
	}

	if oauth.codeVerifier != "verifier" {
		t.Fatalf("expected exchanged verifier, got %q", oauth.codeVerifier)
	}

	sessionCookie := assertCookie(
		t,
		rec.Result(),
		sessionCookieName,
	)

	meReq := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/me",
		nil,
	)
	meReq.AddCookie(sessionCookie)

	meRec := httptest.NewRecorder()

	handler.Me(
		meRec,
		meReq,
	)

	if meRec.Code != http.StatusOK {
		t.Fatalf("expected ok, got %d", meRec.Code)
	}

	if !strings.Contains(meRec.Body.String(), "user-123") {
		t.Fatalf("expected subject in response, got %s", meRec.Body.String())
	}

	if !strings.Contains(meRec.Body.String(), "tester") {
		t.Fatalf("expected preferred username in response, got %s", meRec.Body.String())
	}

	if !strings.Contains(meRec.Body.String(), "Test User") {
		t.Fatalf("expected name in response, got %s", meRec.Body.String())
	}

	if !strings.Contains(meRec.Body.String(), "org-123") {
		t.Fatalf("expected organization id in response, got %s", meRec.Body.String())
	}

	if !strings.Contains(meRec.Body.String(), "raw_claims") {
		t.Fatalf("expected raw claims in response, got %s", meRec.Body.String())
	}
}

func TestCallbackIdentityServiceCalled(t *testing.T) {
	oauth := &mockOAuthProvider{
		token: &identity.TokenResponse{
			IDToken: "id-token",
		},
	}
	validator := &mockTokenValidator{
		claims: &identity.Claims{
			UserID:         "user-123",
			Email:          "test@example.com",
			Name:           "Test User",
			OrganizationID: "org-123",
			Nonce:          "nonce",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(
					time.Now().Add(time.Hour),
				),
			},
		},
	}
	idSvc := &mockIdentityService{}
	handler := newTestHandlerWithIdentity(
		t,
		oauth,
		validator,
		idSvc,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/callback?code=abc&state=stored",
		nil,
	)
	req.AddCookie(testCookie(stateCookieName, "stored"))
	req.AddCookie(testCookie(nonceCookieName, "nonce"))
	req.AddCookie(testCookie(codeVerifierCookieName, "verifier"))

	rec := httptest.NewRecorder()

	handler.Callback(
		rec,
		req,
	)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}

	if idSvc.zitadelUserID != "user-123" {
		t.Fatalf("expected identity user id, got %q", idSvc.zitadelUserID)
	}

	if idSvc.email != "test@example.com" {
		t.Fatalf("expected identity email, got %q", idSvc.email)
	}

	if idSvc.name != "Test User" {
		t.Fatalf("expected identity name, got %q", idSvc.name)
	}
}

func TestCallbackRejectsIdentityServiceError(t *testing.T) {
	oauth := &mockOAuthProvider{
		token: &identity.TokenResponse{
			IDToken: "id-token",
		},
	}
	validator := &mockTokenValidator{
		claims: &identity.Claims{
			UserID: "user-123",
			Email:  "test@example.com",
			Nonce:  "nonce",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(
					time.Now().Add(time.Hour),
				),
			},
		},
	}
	handler := newTestHandlerWithIdentity(
		t,
		oauth,
		validator,
		&mockIdentityService{
			err: errors.New("db error"),
		},
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/callback?code=abc&state=stored",
		nil,
	)
	req.AddCookie(testCookie(stateCookieName, "stored"))
	req.AddCookie(testCookie(nonceCookieName, "nonce"))
	req.AddCookie(testCookie(codeVerifierCookieName, "verifier"))

	rec := httptest.NewRecorder()

	handler.Callback(
		rec,
		req,
	)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected internal server error, got %d", rec.Code)
	}

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			t.Fatal("expected no session cookie when identity service fails")
		}
	}
}

func TestCallbackSucceedsWithoutOrganization(t *testing.T) {
	oauth := &mockOAuthProvider{
		token: &identity.TokenResponse{
			IDToken: "id-token",
		},
	}
	validator := &mockTokenValidator{
		claims: &identity.Claims{
			UserID: "user-new",
			Email:  "new@example.com",
			Name:   "New User",
			Nonce:  "nonce",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(
					time.Now().Add(time.Hour),
				),
			},
		},
	}
	idSvc := &mockIdentityService{}
	handler := newTestHandlerWithIdentity(
		t,
		oauth,
		validator,
		idSvc,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/callback?code=abc&state=stored",
		nil,
	)
	req.AddCookie(testCookie(stateCookieName, "stored"))
	req.AddCookie(testCookie(nonceCookieName, "nonce"))
	req.AddCookie(testCookie(codeVerifierCookieName, "verifier"))

	rec := httptest.NewRecorder()

	handler.Callback(
		rec,
		req,
	)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}

	location := rec.Header().Get("Location")
	if location != "/onboarding" {
		t.Fatalf("expected redirect to /onboarding, got %q", location)
	}

	assertCookie(t, rec.Result(), sessionCookieName)
}

func TestCallbackRedirectsDashboardWithOrganization(t *testing.T) {
	oauth := &mockOAuthProvider{
		token: &identity.TokenResponse{
			IDToken: "id-token",
		},
	}
	validator := &mockTokenValidator{
		claims: &identity.Claims{
			UserID:         "user-123",
			Email:          "test@example.com",
			OrganizationID: "org-123",
			Nonce:          "nonce",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(
					time.Now().Add(time.Hour),
				),
			},
		},
	}
	idSvc := &mockIdentityService{}
	handler := newTestHandlerWithIdentity(
		t,
		oauth,
		validator,
		idSvc,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/callback?code=abc&state=stored",
		nil,
	)
	req.AddCookie(testCookie(stateCookieName, "stored"))
	req.AddCookie(testCookie(nonceCookieName, "nonce"))
	req.AddCookie(testCookie(codeVerifierCookieName, "verifier"))

	rec := httptest.NewRecorder()

	handler.Callback(
		rec,
		req,
	)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}

	location := rec.Header().Get("Location")
	if location != "/dashboard" {
		t.Fatalf("expected redirect to /dashboard, got %q", location)
	}
}

func TestCallbackUpdatesIdentityData(t *testing.T) {
	oauth := &mockOAuthProvider{
		token: &identity.TokenResponse{
			IDToken: "id-token",
		},
	}
	validator := &mockTokenValidator{
		claims: &identity.Claims{
			UserID: "user-123",
			Email:  "newemail@example.com",
			Name:   "Updated Name",
			Nonce:  "nonce",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(
					time.Now().Add(time.Hour),
				),
			},
		},
	}
	idSvc := &mockIdentityService{}
	handler := newTestHandlerWithIdentity(
		t,
		oauth,
		validator,
		idSvc,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/callback?code=abc&state=stored",
		nil,
	)
	req.AddCookie(testCookie(stateCookieName, "stored"))
	req.AddCookie(testCookie(nonceCookieName, "nonce"))
	req.AddCookie(testCookie(codeVerifierCookieName, "verifier"))

	rec := httptest.NewRecorder()

	handler.Callback(
		rec,
		req,
	)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}

	if idSvc.email != "newemail@example.com" {
		t.Fatalf("expected updated email passed to identity service, got %q", idSvc.email)
	}

	if idSvc.name != "Updated Name" {
		t.Fatalf("expected updated name passed to identity service, got %q", idSvc.name)
	}
}

func TestCallbackRejectsNonceMismatch(t *testing.T) {
	handler := newTestHandler(
		t,
		&mockOAuthProvider{
			token: &identity.TokenResponse{
				IDToken: "id-token",
			},
		},
		&mockTokenValidator{
			claims: &identity.Claims{
				UserID: "user-123",
				Nonce:  "different",
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(
						time.Now().Add(time.Hour),
					),
				},
			},
		},
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/callback?code=abc&state=stored",
		nil,
	)
	req.AddCookie(testCookie(stateCookieName, "stored"))
	req.AddCookie(testCookie(nonceCookieName, "nonce"))
	req.AddCookie(testCookie(codeVerifierCookieName, "verifier"))

	rec := httptest.NewRecorder()

	handler.Callback(
		rec,
		req,
	)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", rec.Code)
	}
}

func TestMeRejectsMissingSession(t *testing.T) {
	handler := newTestHandler(
		t,
		&mockOAuthProvider{},
		&mockTokenValidator{},
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/me",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.Me(
		rec,
		req,
	)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d", rec.Code)
	}
}

func TestLogoutClearsSession(t *testing.T) {
	handler := newTestHandler(
		t,
		&mockOAuthProvider{},
		&mockTokenValidator{},
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/logout",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.Logout(
		rec,
		req,
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected ok, got %d", rec.Code)
	}

	cookie := assertCookie(
		t,
		rec.Result(),
		sessionCookieName,
	)

	if cookie.MaxAge != -1 {
		t.Fatalf("expected clearing cookie, got max age %d", cookie.MaxAge)
	}
}

func newTestHandler(
	t *testing.T,
	oauth OAuthProvider,
	validator identity.Provider,
) *Handler {
	t.Helper()

	return newTestHandlerWithIdentity(
		t,
		oauth,
		validator,
		nil,
	)
}

func newTestHandlerWithIdentity(
	t *testing.T,
	oauth OAuthProvider,
	validator identity.Provider,
	identitySvc IdentityService,
) *Handler {
	t.Helper()

	handler, err := NewHandler(
		oauth,
		validator,
		identitySvc,
		"01234567890123456789012345678901",
		false,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		false,
	)

	if err != nil {
		t.Fatalf("create handler: %v", err)
	}

	return handler
}

func testCookie(
	name string,
	value string,
) *http.Cookie {
	return &http.Cookie{
		Name:  name,
		Value: value,
	}
}

func assertCookie(
	t *testing.T,
	resp *http.Response,
	name string,
) *http.Cookie {
	t.Helper()

	for _, cookie := range resp.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}

	t.Fatalf("expected cookie %q", name)

	return nil
}

type mockOAuthProvider struct {
	authorizationURL string
	request          identity.AuthorizationRequest
	token            *identity.TokenResponse
	code             string
	codeVerifier     string
}

func (m *mockOAuthProvider) AuthorizationURL(
	ctx context.Context,
	request identity.AuthorizationRequest,
) (string, error) {
	m.request = request

	if m.authorizationURL != "" {
		return m.authorizationURL, nil
	}

	values := url.Values{}
	values.Set("state", request.State)

	return "http://issuer.example/authorize?" + values.Encode(), nil
}

func (m *mockOAuthProvider) ExchangeCode(
	ctx context.Context,
	code string,
	codeVerifier string,
) (*identity.TokenResponse, error) {
	m.code = code
	m.codeVerifier = codeVerifier

	return m.token, nil
}

type mockTokenValidator struct {
	claims *identity.Claims
}

func (m *mockTokenValidator) ValidateToken(
	ctx context.Context,
	token string,
) (*identity.Claims, error) {
	return m.claims, nil
}

type mockIdentityService struct {
	zitadelUserID string
	email         string
	name          string
	err           error
}

func (m *mockIdentityService) FindOrCreateIdentity(
	ctx context.Context,
	zitadelUserID string,
	email string,
	name string,
) error {
	m.zitadelUserID = zitadelUserID
	m.email = email
	m.name = name
	return m.err
}
