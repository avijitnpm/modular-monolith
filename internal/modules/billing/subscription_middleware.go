package billing

import (
	"net/http"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

func RequireSubscription(checker SubscriptionChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			organizationID, ok := appcontext.GetOrganizationID(r.Context())
			if !ok {
				response.Error(w, http.StatusForbidden, "active subscription required")
				return
			}

			active, err := checker.HasActiveSubscription(r.Context(), organizationID)
			if err != nil || !active {
				response.Error(w, http.StatusForbidden, "active subscription required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
