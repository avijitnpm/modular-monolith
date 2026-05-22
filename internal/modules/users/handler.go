package users

import (
	"encoding/json"
	"net/http"

	"github.com/avijitnpm/modular-monolith/internal/audit"
	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/internal/service"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

type Handler struct {
	Service      *service.Service
	AuditService *audit.Service
}

func NewHandler(
	service *service.Service,
	auditService *audit.Service,
) *Handler {

	return &Handler{
		Service:      service,
		AuditService: auditService,
	}
}

func (h *Handler) RegisterUser(
	w http.ResponseWriter,
	r *http.Request,
) {

	authenticatedUser, ok := appcontext.GetAuthenticatedUser(
		r.Context(),
	)

	if !ok {

		response.InternalServerError(
			w,
			"authenticated user missing",
		)

		return
	}

	organizationID, organizationOK := appcontext.GetOrganizationID(
		r.Context(),
	)

	if !organizationOK {

		response.InternalServerError(
			w,
			"organization context missing",
		)

		return
	}

	var req RegisterUserRequest

	err := json.NewDecoder(r.Body).Decode(
		&req,
	)

	if err != nil {

		response.BadRequest(
			w,
			"invalid request body",
		)

		return
	}

	user, err := h.Service.RegisterUser(
		r.Context(),
		organizationID,
		req.ZitadelUserID,
		req.Email,
	)

	if err != nil {

		response.InternalServerError(
			w,
			"failed to register user",
		)

		return
	}

	err = h.AuditService.Log(
		r.Context(),
		&audit.Event{
			OrganizationID: organizationID,
			UserID:         authenticatedUser.UserID,
			Action:         "user_registered",
			EntityType:     "user",
			EntityID:       user.ID,
		},
	)

	if err != nil {

		response.InternalServerError(
			w,
			"failed to create audit log",
		)

		return
	}

	response.JSON(
		w,
		http.StatusCreated,
		map[string]interface{}{
			"data": map[string]interface{}{
				"id":    user.ID,
				"email": user.Email,
			},
		},
	)
}
