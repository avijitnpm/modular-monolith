package entitlements

import "context"

type SubscriptionGetter interface {
	GetSubscription(ctx context.Context, organizationID string) (plan string, status string, err error)
}

type UsageGetter interface {
	GetUsage(ctx context.Context, organizationID string, metric string) (int64, error)
}

type Service struct {
	Subscriptions SubscriptionGetter
	Usage         UsageGetter
}

func NewService(subscriptions SubscriptionGetter, usage UsageGetter) *Service {
	return &Service{Subscriptions: subscriptions, Usage: usage}
}

func (s *Service) resolvePlan(ctx context.Context, organizationID string) string {
	plan, status, err := s.Subscriptions.GetSubscription(ctx, organizationID)
	if err != nil || plan == "" {
		return "free"
	}
	if status == "active" || status == "trialing" {
		return plan
	}
	return "free"
}

func (s *Service) GetLimit(ctx context.Context, organizationID string, metric string) (int64, error) {
	plan := s.resolvePlan(ctx, organizationID)
	return limitForPlan(plan, metric), nil
}

func (s *Service) GetUsage(ctx context.Context, organizationID string, metric string) (int64, error) {
	return s.Usage.GetUsage(ctx, organizationID, metric)
}

func (s *Service) CanUse(ctx context.Context, organizationID string, metric string, amount int64) (*Entitlement, error) {
	plan := s.resolvePlan(ctx, organizationID)
	limit := limitForPlan(plan, metric)

	used, err := s.Usage.GetUsage(ctx, organizationID, metric)
	if err != nil {
		return nil, err
	}

	e := &Entitlement{
		Metric: metric,
		Used:   used,
		Limit:  limit,
	}

	if limit == Unlimited {
		e.Remaining = Unlimited
		e.Allowed = true
	} else {
		e.Remaining = limit - used
		if e.Remaining < 0 {
			e.Remaining = 0
		}
		e.Allowed = used+amount <= limit
	}

	return e, nil
}

func (s *Service) GetEntitlements(ctx context.Context, organizationID string) ([]Entitlement, error) {
	result := make([]Entitlement, 0, len(metrics))
	plan := s.resolvePlan(ctx, organizationID)

	for _, metric := range metrics {
		limit := limitForPlan(plan, metric)
		used, err := s.Usage.GetUsage(ctx, organizationID, metric)
		if err != nil {
			return nil, err
		}

		e := Entitlement{
			Metric: metric,
			Used:   used,
			Limit:  limit,
		}

		if limit == Unlimited {
			e.Remaining = Unlimited
			e.Allowed = true
		} else {
			e.Remaining = limit - used
			if e.Remaining < 0 {
				e.Remaining = 0
			}
			e.Allowed = used < limit
		}

		result = append(result, e)
	}

	return result, nil
}
