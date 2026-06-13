package billing

import "time"

type CreateSubscriptionRequest struct {
	Provider string `json:"provider"`
	Plan     string `json:"plan"`
	Status   string `json:"status"`
}

type CreateCheckoutRequest struct {
	Plan string `json:"plan"`
}

type UpdateSubscriptionRequest struct {
	Plan             string     `json:"plan"`
	Status           string     `json:"status"`
	CurrentPeriodEnd *time.Time `json:"current_period_end"`
}

type CheckoutResponse struct {
	CheckoutURL string `json:"checkout_url"`
}

type SubscriptionResponse struct {
	ID                     string     `json:"id"`
	OrganizationID         string     `json:"organization_id"`
	Provider               string     `json:"provider"`
	ProviderCustomerID     *string    `json:"provider_customer_id"`
	ProviderSubscriptionID *string    `json:"provider_subscription_id"`
	Plan                   string     `json:"plan"`
	Status                 string     `json:"status"`
	CurrentPeriodEnd       *time.Time `json:"current_period_end"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

func checkoutResponse(
	url string,
) CheckoutResponse {

	return CheckoutResponse{
		CheckoutURL: url,
	}
}

func subscriptionResponse(
	subscription *Subscription,
) *SubscriptionResponse {

	if subscription == nil {
		return nil
	}

	return &SubscriptionResponse{
		ID:                     subscription.ID,
		OrganizationID:         subscription.OrganizationID,
		Provider:               subscription.Provider,
		ProviderCustomerID:     subscription.ProviderCustomerID,
		ProviderSubscriptionID: subscription.ProviderSubscriptionID,
		Plan:                   subscription.Plan,
		Status:                 subscription.Status,
		CurrentPeriodEnd:       subscription.CurrentPeriodEnd,
		CreatedAt:              subscription.CreatedAt,
		UpdatedAt:              subscription.UpdatedAt,
	}
}
