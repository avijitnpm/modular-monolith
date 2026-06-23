package billing

import "context"

type SubscriptionChecker interface {
	HasActiveSubscription(ctx context.Context, organizationID string) (bool, error)
	IsPlan(ctx context.Context, organizationID string, plan string) (bool, error)
}

type subscriptionChecker struct {
	store Store
}

func NewSubscriptionChecker(store Store) SubscriptionChecker {
	return &subscriptionChecker{store: store}
}

func (c *subscriptionChecker) HasActiveSubscription(ctx context.Context, organizationID string) (bool, error) {
	sub, err := c.store.GetSubscription(ctx, organizationID)
	if err != nil {
		return false, err
	}
	if sub == nil {
		return false, nil
	}
	return sub.Status == "active" || sub.Status == "trialing", nil
}

func (c *subscriptionChecker) IsPlan(ctx context.Context, organizationID string, plan string) (bool, error) {
	sub, err := c.store.GetSubscription(ctx, organizationID)
	if err != nil {
		return false, err
	}
	if sub == nil {
		return false, nil
	}
	return sub.Plan == plan, nil
}
