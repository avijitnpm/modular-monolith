package billing

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/platform/payments"
)

type Store interface {
	GetSubscription(
		ctx context.Context,
		organizationID string,
	) (*Subscription, error)
	CreateSubscription(
		ctx context.Context,
		organizationID string,
		provider string,
		plan string,
		status string,
	) (*Subscription, error)
	UpdateSubscription(
		ctx context.Context,
		id string,
		organizationID string,
		plan string,
		status string,
		currentPeriodEnd *time.Time,
	) (*Subscription, error)
}

type Service struct {
	Repository Store
	Provider   payments.Provider
}

func NewService(
	repository Store,
	provider payments.Provider,
) *Service {

	return &Service{
		Repository: repository,
		Provider:   provider,
	}
}

func (s *Service) GetSubscription(
	ctx context.Context,
	organizationID string,
) (*Subscription, error) {

	return s.Repository.GetSubscription(
		ctx,
		organizationID,
	)
}

func (s *Service) CreateCheckoutSession(
	ctx context.Context,
	organizationID string,
	plan string,
) (string, error) {

	organizationID = strings.TrimSpace(organizationID)
	plan = strings.TrimSpace(plan)

	if organizationID == "" || plan == "" {
		return "", ErrInvalidSubscription
	}

	if s.Provider == nil {
		return "", errors.New("checkout provider is not configured")
	}

	return s.Provider.CreateCheckoutSession(
		ctx,
		organizationID,
		plan,
	)
}

func (s *Service) CreateSubscription(
	ctx context.Context,
	organizationID string,
	provider string,
	plan string,
	status string,
) (*Subscription, error) {

	provider = strings.TrimSpace(provider)
	plan = strings.TrimSpace(plan)
	status = strings.TrimSpace(status)

	if provider == "" || plan == "" || status == "" {
		return nil, ErrInvalidSubscription
	}

	return s.Repository.CreateSubscription(
		ctx,
		organizationID,
		provider,
		plan,
		status,
	)
}

func (s *Service) UpdateSubscription(
	ctx context.Context,
	id string,
	organizationID string,
	plan string,
	status string,
	currentPeriodEnd *time.Time,
) (*Subscription, error) {

	id = strings.TrimSpace(id)
	plan = strings.TrimSpace(plan)
	status = strings.TrimSpace(status)

	if id == "" || plan == "" || status == "" {
		return nil, ErrInvalidSubscription
	}

	return s.Repository.UpdateSubscription(
		ctx,
		id,
		organizationID,
		plan,
		status,
		currentPeriodEnd,
	)
}
