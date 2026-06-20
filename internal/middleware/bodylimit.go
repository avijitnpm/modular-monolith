package middleware

import (
	"net/http"

	"github.com/avijitnpm/modular-monolith/pkg/response"
)

// BodyLimit restricts request body size to maxBytes.
func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)

			// MaxBytesReader will cause handlers to get an error when
			// reading beyond the limit; handlers already return appropriate errors.
			// If the error hasn't been handled yet, the 413 is surfaced by the
			// response package or standard library.
		})
	}
}

// BodyLimitError is a helper to detect and respond to body-too-large errors.
func BodyLimitError(w http.ResponseWriter, err error) bool {
	if err != nil && err.Error() == "http: request body too large" {
		response.Error(w, http.StatusRequestEntityTooLarge, "request body too large")
		return true
	}
	return false
}
