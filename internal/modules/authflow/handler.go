package authflow

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/providers/identity"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

const (
	stateCookieName        = "mm_oauth_state"
	nonceCookieName        = "mm_oauth_nonce"
	codeVerifierCookieName = "mm_oauth_code_verifier"

	oauthCookieTTL = 10 * time.Minute
	sessionTTL     = 24 * time.Hour
)

type OAuthProvider interface {
	AuthorizationURL(
		ctx context.Context,
		request identity.AuthorizationRequest,
	) (string, error)

	ExchangeCode(
		ctx context.Context,
		code string,
		codeVerifier string,
	) (*identity.TokenResponse, error)
}

type UserProvisioner interface {
	ProvisionAuthenticatedUser(
		ctx context.Context,
		zitadelUserID string,
		email string,
		organizationID string,
	) error
}

type Handler struct {
	oauth           OAuthProvider
	tokenValidator  identity.Provider
	userProvisioner UserProvisioner
	sessions        *sessionManager
	logger          *slog.Logger
	secureCookies   bool
	logClaimKeys    bool
}

var discoveredClaimKeysOnce sync.Once

func NewHandler(
	oauth OAuthProvider,
	tokenValidator identity.Provider,
	userProvisioner UserProvisioner,
	sessionSecret string,
	secureCookies bool,
	logger *slog.Logger,
	logClaimKeys bool,
) (*Handler, error) {
	sessions, err := newSessionManager(
		sessionSecret,
		secureCookies,
	)

	if err != nil {
		return nil, err
	}

	return &Handler{
		oauth:           oauth,
		tokenValidator:  tokenValidator,
		userProvisioner: userProvisioner,
		sessions:        sessions,
		logger:          logger,
		secureCookies:   secureCookies,
		logClaimKeys:    logClaimKeys,
	}, nil
}

func (h *Handler) Login(
	w http.ResponseWriter,
	r *http.Request,
) {
	state, err := randomToken()

	if err != nil {
		response.InternalServerError(
			w,
			"failed to start login",
		)

		return
	}

	nonce, err := randomToken()

	if err != nil {
		response.InternalServerError(
			w,
			"failed to start login",
		)

		return
	}

	codeVerifier, err := randomToken()

	if err != nil {
		response.InternalServerError(
			w,
			"failed to start login",
		)

		return
	}

	authURL, err := h.oauth.AuthorizationURL(
		r.Context(),
		identity.AuthorizationRequest{
			State:         state,
			Nonce:         nonce,
			CodeChallenge: codeChallenge(codeVerifier),
			Scope:         "openid email profile",
		},
	)

	if err != nil {
		response.InternalServerError(
			w,
			"failed to start login",
		)

		return
	}

	h.setOAuthCookie(
		w,
		stateCookieName,
		state,
	)
	h.setOAuthCookie(
		w,
		nonceCookieName,
		nonce,
	)
	h.setOAuthCookie(
		w,
		codeVerifierCookieName,
		codeVerifier,
	)

	http.Redirect(
		w,
		r,
		authURL,
		http.StatusFound,
	)
}

func (h *Handler) Callback(
	w http.ResponseWriter,
	r *http.Request,
) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		h.logger.Info("callback step", "step", "missing_code_or_state")
		response.BadRequest(
			w,
			"invalid auth callback",
		)

		return
	}

	storedState, err := h.oauthCookie(
		r,
		stateCookieName,
	)

	if err != nil || storedState != state {
		h.logger.Info("callback step", "step", "state_mismatch")
		response.BadRequest(
			w,
			"invalid auth callback",
		)

		return
	}

	codeVerifier, err := h.oauthCookie(
		r,
		codeVerifierCookieName,
	)

	if err != nil {
		h.logger.Info("callback step", "step", "missing_code_verifier")
		response.BadRequest(
			w,
			"invalid auth callback",
		)

		return
	}

	nonce, err := h.oauthCookie(
		r,
		nonceCookieName,
	)

	if err != nil {
		h.logger.Info("callback step", "step", "missing_nonce")
		response.BadRequest(
			w,
			"invalid auth callback",
		)

		return
	}

	token, err := h.oauth.ExchangeCode(
		r.Context(),
		code,
		codeVerifier,
	)

	if err != nil {
		h.logger.Info("callback step", "step", "token_exchange_failed", "error", err)
		response.BadRequest(
			w,
			"invalid auth callback",
		)

		return
	}

	h.logger.Info("callback step", "step", "token_exchange_success")

	claims, err := h.tokenValidator.ValidateToken(
		r.Context(),
		token.IDToken,
	)

	if err != nil {
		h.logger.Info("callback step", "step", "id_token_validation_failed", "error", err)
		response.BadRequest(
			w,
			"invalid auth callback",
		)

		return
	}

	h.logger.Info("callback step", "step", "id_token_validated")

	if claims.Nonce != nonce {
		h.logger.Info("callback step", "step", "nonce_mismatch")
		response.BadRequest(
			w,
			"invalid auth callback",
		)

		return
	}

	h.logDiscoveredClaimKeys(claims.RawClaims)

	user := normalizeUser(claims)

	h.logger.Info("callback step", "step", "user_claims_extracted",
		"subject", user.Subject,
		"email", user.Email,
		"organization_id", user.OrganizationID,
	)

	if h.userProvisioner != nil {
		h.logger.Info("callback step", "step", "provisioning_started")

		err = h.userProvisioner.ProvisionAuthenticatedUser(
			r.Context(),
			user.Subject,
			user.Email,
			user.OrganizationID,
		)

		if err != nil {
			h.logger.Error("user provisioning failed",
				"error", err,
				"subject", user.Subject,
				"email", user.Email,
				"organization_id", user.OrganizationID,
			)
			response.InternalServerError(
				w,
				"failed to provision user",
			)

			return
		}

		h.logger.Info("callback step", "step", "provisioning_completed")
	}

	expiresAt := time.Now().Add(sessionTTL)

	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	err = h.sessions.set(
		w,
		Session{
			User:      user,
			ExpiresAt: expiresAt.Unix(),
		},
	)

	if err != nil {
		h.logger.Info("callback step", "step", "session_creation_failed", "error", err)
		response.InternalServerError(
			w,
			"failed to create session",
		)

		return
	}

	h.clearOAuthCookie(
		w,
		stateCookieName,
	)
	h.clearOAuthCookie(
		w,
		nonceCookieName,
	)
	h.clearOAuthCookie(
		w,
		codeVerifierCookieName,
	)

	h.logger.Info("callback step", "step", "login_complete")

	http.Redirect(
		w,
		r,
		"/dashboard",
		http.StatusFound,
	)
}

func (h *Handler) logDiscoveredClaimKeys(
	claims map[string]any,
) {
	if !h.logClaimKeys || h.logger == nil || len(claims) == 0 {
		return
	}

	discoveredClaimKeysOnce.Do(func() {
		keys := make(
			[]string,
			0,
			len(claims),
		)

		for key := range claims {
			keys = append(
				keys,
				key,
			)
		}

		sort.Strings(keys)

		h.logger.Debug(
			"authenticated token claim keys discovered",
			"claim_keys",
			keys,
		)
	})
}

func (h *Handler) Logout(
	w http.ResponseWriter,
	r *http.Request,
) {
	h.sessions.clear(w)

	response.OK(
		w,
		map[string]bool{
			"authenticated": false,
		},
	)
}

func (h *Handler) Me(
	w http.ResponseWriter,
	r *http.Request,
) {
	session, err := h.sessions.get(r)

	if err != nil {
		response.Error(
			w,
			http.StatusUnauthorized,
			"not authenticated",
		)

		return
	}

	response.OK(
		w,
		map[string]any{
			"authenticated": true,
			"user":          session.User,
		},
	)
}

func (h *Handler) setOAuthCookie(
	w http.ResponseWriter,
	name string,
	value string,
) {
	http.SetCookie(
		w,
		&http.Cookie{
			Name:     name,
			Value:    value,
			Path:     "/api/v1/auth",
			Expires:  time.Now().Add(oauthCookieTTL),
			MaxAge:   int(oauthCookieTTL.Seconds()),
			HttpOnly: true,
			Secure:   h.secureCookies,
			SameSite: http.SameSiteLaxMode,
		},
	)
}

func (h *Handler) oauthCookie(
	r *http.Request,
	name string,
) (string, error) {
	cookie, err := r.Cookie(name)

	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}

func (h *Handler) clearOAuthCookie(
	w http.ResponseWriter,
	name string,
) {
	http.SetCookie(
		w,
		&http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/api/v1/auth",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   h.secureCookies,
			SameSite: http.SameSiteLaxMode,
		},
	)
}
