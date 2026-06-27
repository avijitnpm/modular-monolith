package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"time"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
)

// SensitiveHeaders must never be logged. If header logging is added
// in the future, filter these out.
var SensitiveHeaders = map[string]struct{}{
	"Authorization": {},
	"Cookie":        {},
	"Set-Cookie":    {},
}

// loggingWriter wraps http.ResponseWriter to capture the status code for logging.
type loggingWriter struct {
	http.ResponseWriter
	status int
}

func (w *loggingWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *loggingWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func (w *loggingWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			start := time.Now()

			lw := &loggingWriter{ResponseWriter: w}
			next.ServeHTTP(lw, r)

			status := lw.status
			if status == 0 {
				status = http.StatusOK
			}

			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"duration", time.Since(start).String(),
				"client_ip", extractClientIP(r),
				"user_agent", r.UserAgent(),
			}

			if rid, ok := r.Context().Value(RequestIDKey).(string); ok && rid != "" {
				attrs = append(attrs, "request_id", rid)
			}

			if id, ok := appcontext.GetIdentity(r.Context()); ok && id.IdentityID != "" {
				attrs = append(attrs, "identity", id.IdentityID)
			}

			if orgID, ok := appcontext.GetOrganizationID(r.Context()); ok && orgID != "" {
				attrs = append(attrs, "org_id", orgID)
			}

			logger.InfoContext(r.Context(), "http request", attrs...)
		})
	}
}

func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
