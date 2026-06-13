package billing

import (
	"encoding/json"
	"errors"
	"net/http"

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

func (h *Handler) GetBilling(
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

	subscription, err := h.Service.GetSubscription(
		r.Context(),
		organizationID,
	)

	if err != nil {
		response.InternalServerError(
			w,
			"failed to get billing",
		)

		return
	}

	response.OK(
		w,
		subscriptionResponse(subscription),
	)
}

func (h *Handler) CreateBilling(
	w http.ResponseWriter,
	r *http.Request,
) {

	organizationID, ok := organizationIDFromRequest(
		w,
		r,
	)

	if !ok {
		return
	}

	var req CreateSubscriptionRequest

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

	subscription, err := h.Service.CreateSubscription(
		r.Context(),
		organizationID,
		req.Provider,
		req.Plan,
		req.Status,
	)

	if errors.Is(err, ErrInvalidSubscription) {
		response.BadRequest(
			w,
			"provider, plan, and status are required",
		)

		return
	}

	if errors.Is(err, ErrSubscriptionAlreadyExists) {
		response.Error(
			w,
			http.StatusConflict,
			"subscription already exists",
		)

		return
	}

	if err != nil {
		response.InternalServerError(
			w,
			"failed to create billing",
		)

		return
	}

	response.Created(
		w,
		subscriptionResponse(subscription),
	)
}

func (h *Handler) CreateCheckout(
	w http.ResponseWriter,
	r *http.Request,
) {

	organizationID, ok := organizationIDFromRequest(
		w,
		r,
	)

	if !ok {
		return
	}

	var req CreateCheckoutRequest

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

	checkoutURL, err := h.Service.CreateCheckoutSession(
		r.Context(),
		organizationID,
		req.Plan,
	)

	if errors.Is(err, ErrInvalidSubscription) {
		response.BadRequest(
			w,
			"plan is required",
		)

		return
	}

	if err != nil {
		response.InternalServerError(
			w,
			"failed to create checkout session",
		)

		return
	}

	response.OK(
		w,
		checkoutResponse(checkoutURL),
	)
}

func (h *Handler) UpdateBilling(
	w http.ResponseWriter,
	r *http.Request,
) {

	organizationID, ok := organizationIDFromRequest(
		w,
		r,
	)

	if !ok {
		return
	}

	subscriptionID := chi.URLParam(
		r,
		"id",
	)

	var req UpdateSubscriptionRequest

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

	subscription, err := h.Service.UpdateSubscription(
		r.Context(),
		subscriptionID,
		organizationID,
		req.Plan,
		req.Status,
		req.CurrentPeriodEnd,
	)

	if errors.Is(err, ErrInvalidSubscription) {
		response.BadRequest(
			w,
			"plan and status are required",
		)

		return
	}

	if errors.Is(err, ErrSubscriptionNotFound) {
		response.Error(
			w,
			http.StatusNotFound,
			"subscription not found",
		)

		return
	}

	if err != nil {
		response.InternalServerError(
			w,
			"failed to update billing",
		)

		return
	}

	response.OK(
		w,
		subscriptionResponse(subscription),
	)
}

func organizationIDFromRequest(
	w http.ResponseWriter,
	r *http.Request,
) (string, bool) {

	organizationID, ok := appcontext.GetOrganizationID(
		r.Context(),
	)

	if !ok {
		response.InternalServerError(
			w,
			"organization context missing",
		)

		return "", false
	}

	return organizationID, true
}
