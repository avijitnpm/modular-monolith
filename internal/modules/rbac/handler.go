package rbac

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	appcontext "github.com/avijitnpm/modular-monolith/internal/context"
	"github.com/avijitnpm/modular-monolith/pkg/response"
)

type Handler struct {
	Service *Service
}

func NewHandler(
	service *Service,
) *Handler {

	return &Handler{
		Service: service,
	}
}

func (h *Handler) ListRoles(
	w http.ResponseWriter,
	r *http.Request,
) {

	organizationID, ok := appcontext.GetOrganizationID(
		r.Context(),
	)

	if !ok {
		response.InternalServerError(
			w,
			"organization context missing",
		)

		return
	}

	roles, err := h.Service.ListRoles(
		r.Context(),
		organizationID,
	)

	if err != nil {
		response.InternalServerError(
			w,
			"failed to list roles",
		)

		return
	}

	payload := make(
		[]RoleResponse,
		0,
		len(roles),
	)

	for _, role := range roles {
		payload = append(
			payload,
			roleResponse(role),
		)
	}

	response.OK(
		w,
		payload,
	)
}

func (h *Handler) CreateRole(
	w http.ResponseWriter,
	r *http.Request,
) {

	authenticatedUser, organizationID, ok := requestIdentity(
		w,
		r,
	)

	if !ok {
		return
	}

	var req CreateRoleRequest

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

	if strings.TrimSpace(req.Name) == "" {
		response.BadRequest(
			w,
			"role name is required",
		)

		return
	}

	role, err := h.Service.CreateRole(
		r.Context(),
		organizationID,
		authenticatedUser.UserID,
		req.Name,
		req.Permissions,
	)

	if errors.Is(err, ErrRoleAlreadyExists) {
		response.BadRequest(
			w,
			"role already exists",
		)

		return
	}

	if errors.Is(err, ErrUnknownPermissions) {
		response.BadRequest(
			w,
			"unknown permissions",
		)

		return
	}

	if err != nil {
		response.InternalServerError(
			w,
			"failed to create role",
		)

		return
	}

	response.Created(
		w,
		roleResponse(*role),
	)
}

func (h *Handler) ListPermissions(
	w http.ResponseWriter,
	r *http.Request,
) {

	permissions, err := h.Service.ListPermissions(
		r.Context(),
	)

	if err != nil {
		response.InternalServerError(
			w,
			"failed to list permissions",
		)

		return
	}

	payload := make(
		[]PermissionResponse,
		0,
		len(permissions),
	)

	for _, permission := range permissions {
		payload = append(
			payload,
			permissionResponse(permission),
		)
	}

	response.OK(
		w,
		payload,
	)
}

func (h *Handler) AssignRoleToUser(
	w http.ResponseWriter,
	r *http.Request,
) {

	authenticatedUser, organizationID, ok := requestIdentity(
		w,
		r,
	)

	if !ok {
		return
	}

	userID := chi.URLParam(
		r,
		"id",
	)

	if userID == "" {
		response.BadRequest(
			w,
			"user id is required",
		)

		return
	}

	var req AssignUserRoleRequest

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

	if strings.TrimSpace(req.RoleID) == "" {
		response.BadRequest(
			w,
			"role id is required",
		)

		return
	}

	assignment, err := h.Service.AssignRoleToUser(
		r.Context(),
		organizationID,
		authenticatedUser.UserID,
		userID,
		req.RoleID,
	)

	if errors.Is(err, ErrUserRoleExists) {
		response.BadRequest(
			w,
			"user role already exists",
		)

		return
	}

	if errors.Is(err, ErrRoleNotFound) {
		response.BadRequest(
			w,
			"role not found",
		)

		return
	}

	if err != nil {
		response.InternalServerError(
			w,
			"failed to assign role",
		)

		return
	}

	response.Created(
		w,
		map[string]string{
			"id":      assignment.ID,
			"user_id": assignment.UserID,
			"role_id": assignment.RoleID,
		},
	)
}

func requestIdentity(
	w http.ResponseWriter,
	r *http.Request,
) (*appcontext.AuthenticatedUser, string, bool) {

	authenticatedUser, ok := appcontext.GetAuthenticatedUser(
		r.Context(),
	)

	if !ok {
		response.InternalServerError(
			w,
			"authenticated user missing",
		)

		return nil, "", false
	}

	organizationID, ok := appcontext.GetOrganizationID(
		r.Context(),
	)

	if !ok {
		response.InternalServerError(
			w,
			"organization context missing",
		)

		return nil, "", false
	}

	return authenticatedUser, organizationID, true
}
