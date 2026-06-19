package auditmod

import (
	"context"
	"net/http"
	"strconv"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/internal/repository"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

type AuditService interface {
	List(
		ctx context.Context,
		organizationID string,
		limit int,
		offset int,
	) ([]repository.AuditLog, error)
}

type Handler struct {
	Service AuditService
}

func NewHandler(service AuditService) *Handler {
	return &Handler{Service: service}
}

type auditLogResponse struct {
	ID         string            `json:"id"`
	Action     string            `json:"action"`
	EntityType string            `json:"entity_type"`
	EntityID   *string           `json:"entity_id"`
	UserID     *string           `json:"user_id"`
	CreatedAt  string            `json:"created_at"`
	Metadata   map[string]string `json:"metadata"`
}

func (h *Handler) ListAuditLogs(
	w http.ResponseWriter,
	r *http.Request,
) {

	organizationID, ok := appcontext.GetOrganizationID(r.Context())

	if !ok {
		response.InternalServerError(w, "organization context missing")
		return
	}

	limit := queryInt(r, "limit", 50)
	offset := queryInt(r, "offset", 0)

	logs, err := h.Service.List(r.Context(), organizationID, limit, offset)

	if err != nil {
		response.InternalServerError(w, "failed to list audit logs")
		return
	}

	result := make([]auditLogResponse, 0, len(logs))
	for _, l := range logs {
		result = append(result, auditLogResponse{
			ID:         l.ID,
			Action:     l.Action,
			EntityType: l.EntityType,
			EntityID:   l.EntityID,
			UserID:     l.UserID,
			CreatedAt:  l.CreatedAt,
			Metadata:   l.Metadata,
		})
	}

	response.OK(w, result)
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return defaultVal
	}
	return n
}
