package authflow

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const sessionCookieName = "mm_session"

var errInvalidSession = errors.New("invalid session")

type Session struct {
	User      SessionUser `json:"user"`
	ExpiresAt int64       `json:"expires_at"`
}

type sessionManager struct {
	aead   cipher.AEAD
	secure bool
}

func newSessionManager(
	secret string,
	secure bool,
) (*sessionManager, error) {
	sum := sha256.Sum256([]byte(secret))

	block, err := aes.NewCipher(sum[:])

	if err != nil {
		return nil, fmt.Errorf("create session cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)

	if err != nil {
		return nil, fmt.Errorf("create session aead: %w", err)
	}

	return &sessionManager{
		aead:   aead,
		secure: secure,
	}, nil
}

func (m *sessionManager) set(
	w http.ResponseWriter,
	session Session,
) error {
	payload, err := json.Marshal(session)

	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	nonce := make([]byte, m.aead.NonceSize())

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("generate session nonce: %w", err)
	}

	sealed := m.aead.Seal(
		nonce,
		nonce,
		payload,
		nil,
	)

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     sessionCookieName,
			Value:    base64.RawURLEncoding.EncodeToString(sealed),
			Path:     "/",
			Expires:  time.Unix(session.ExpiresAt, 0),
			MaxAge:   int(time.Until(time.Unix(session.ExpiresAt, 0)).Seconds()),
			HttpOnly: true,
			Secure:   m.secure,
			SameSite: http.SameSiteLaxMode,
		},
	)

	return nil
}

func (m *sessionManager) get(
	r *http.Request,
) (*Session, error) {
	cookie, err := r.Cookie(sessionCookieName)

	if err != nil {
		return nil, errInvalidSession
	}

	sealed, err := base64.RawURLEncoding.DecodeString(cookie.Value)

	if err != nil {
		return nil, errInvalidSession
	}

	nonceSize := m.aead.NonceSize()

	if len(sealed) <= nonceSize {
		return nil, errInvalidSession
	}

	payload, err := m.aead.Open(
		nil,
		sealed[:nonceSize],
		sealed[nonceSize:],
		nil,
	)

	if err != nil {
		return nil, errInvalidSession
	}

	var session Session

	if err := json.Unmarshal(payload, &session); err != nil {
		return nil, errInvalidSession
	}

	if time.Now().Unix() >= session.ExpiresAt {
		return nil, errInvalidSession
	}

	return &session, nil
}

func (m *sessionManager) clear(
	w http.ResponseWriter,
) {
	http.SetCookie(
		w,
		&http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   m.secure,
			SameSite: http.SameSiteLaxMode,
		},
	)
}
