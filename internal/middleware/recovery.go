package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/avijitnpm/modular-monolith/pkg/logger"
)

func Recovery(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			defer func() {
				if err := recover(); err != nil {

					log.ErrorContext(
						r.Context(),
						"panic recovered",
						"error", logger.RedactString(fmt.Sprint(err)),
						"stack", string(debug.Stack()),
					)

					http.Error(
						w,
						"internal server error",
						http.StatusInternalServerError,
					)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
