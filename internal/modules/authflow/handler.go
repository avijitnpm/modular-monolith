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

type Handler struct {
	oauth          OAuthProvider
	tokenValidator identity.Provider
	sessions       *sessionManager
	logger         *slog.Logger
	secureCookies  bool
	logClaimKeys   bool
}

var discoveredClaimKeysOnce sync.Once

func NewHandler(
	oauth OAuthProvider,
	tokenValidator identity.Provider,
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
		oauth:          oauth,
		tokenValidator: tokenValidator,
		sessions:       sessions,
		logger:         logger,
		secureCookies:  secureCookies,
		logClaimKeys:   logClaimKeys,
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
		response.BadRequest(
			w,
			"invalid auth callback",
		)

		return
	}

	claims, err := h.tokenValidator.ValidateToken(
		r.Context(),
		token.IDToken,
	)

	if err != nil {
		response.BadRequest(
			w,
			"invalid auth callback",
		)

		return
	}

	if claims.Nonce != nonce {
		response.BadRequest(
			w,
			"invalid auth callback",
		)

		return
	}

	h.logDiscoveredClaimKeys(claims.RawClaims)

	expiresAt := time.Now().Add(sessionTTL)

	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	err = h.sessions.set(
		w,
		Session{
			User:      normalizeUser(claims),
			ExpiresAt: expiresAt.Unix(),
		},
	)

	if err != nil {
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
