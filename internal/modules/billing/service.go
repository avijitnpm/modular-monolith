package billing

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/avijitnpm/modular-monolith/internal/audit"
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
	UpsertSubscriptionByProvider(
		ctx context.Context,
		organizationID string,
		provider string,
		providerSubscriptionID string,
		providerCustomerID string,
		plan string,
		status string,
		currentPeriodEnd *time.Time,
	) error
}

type AuditLogger interface {
	Log(ctx context.Context, event *audit.Event) error
}

type Service struct {
	Repository Store
	Provider   payments.Provider
	Audit      AuditLogger
}

func NewService(
	repository Store,
	provider payments.Provider,
	auditLogger AuditLogger,
) *Service {

	return &Service{
		Repository: repository,
		Provider:   provider,
		Audit:      auditLogger,
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

	url, err := s.Provider.CreateCheckoutSession(
		ctx,
		organizationID,
		plan,
	)

	if err != nil {
		return "", err
	}

	s.logAudit(ctx, organizationID, "checkout.created", "", map[string]string{
		"organization_id": organizationID,
		"plan":            plan,
	})

	return url, nil
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

	subscription, err := s.Repository.CreateSubscription(
		ctx,
		organizationID,
		provider,
		plan,
		status,
	)

	if err != nil {
		return nil, err
	}

	s.logAudit(ctx, subscription.OrganizationID, "subscription.created", subscription.ID, map[string]string{
		"organization_id": organizationID,
		"provider":        provider,
		"plan":            plan,
		"status":          status,
	})

	return subscription, nil
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

	existing, _ := s.Repository.GetSubscription(ctx, organizationID)
	previousStatus := ""
	if existing != nil {
		previousStatus = existing.Status
	}

	subscription, err := s.Repository.UpdateSubscription(
		ctx,
		id,
		organizationID,
		plan,
		status,
		currentPeriodEnd,
	)

	if err != nil {
		return nil, err
	}

	s.logAudit(ctx, organizationID, "subscription.updated", id, map[string]string{
		"organization_id": organizationID,
		"previous_status": previousStatus,
		"new_status":      status,
		"plan":            plan,
	})

	return subscription, nil
}

func (s *Service) ProcessWebhookEvent(
	ctx context.Context,
	organizationID string,
	providerSubscriptionID string,
	providerCustomerID string,
	plan string,
	status string,
	currentPeriodEnd *time.Time,
) error {

	if organizationID == "" || status == "" {
		return ErrInvalidSubscription
	}

	existing, _ := s.Repository.GetSubscription(ctx, organizationID)

	previousStatus := ""
	if existing != nil {
		previousStatus = existing.Status
		if previousStatus == status {
			return nil
		}
	}

	err := s.Repository.UpsertSubscriptionByProvider(
		ctx,
		organizationID,
		"dodo",
		providerSubscriptionID,
		providerCustomerID,
		plan,
		status,
		currentPeriodEnd,
	)

	if err != nil {
		return err
	}

	if action := lifecycleAction(status); action != "" {
		s.logAudit(ctx, organizationID, action, providerSubscriptionID, map[string]string{
			"organization_id":          organizationID,
			"previous_status":          previousStatus,
			"new_status":               status,
			"provider_subscription_id": providerSubscriptionID,
		})
	}

	return nil
}

func lifecycleAction(status string) string {
	switch status {
	case "active":
		return "subscription.activated"
	case "past_due":
		return "subscription.past_due"
	case "cancelled":
		return "subscription.cancelled"
	default:
		return ""
	}
}

func (s *Service) logAudit(
	ctx context.Context,
	organizationID string,
	action string,
	entityID string,
	metadata map[string]string,
) {

	if s.Audit == nil {
		return
	}

	_ = s.Audit.Log(ctx, &audit.Event{
		OrganizationID: organizationID,
		Action:         action,
		EntityType:     "subscription",
		EntityID:       entityID,
		Metadata:       metadata,
	})
}
