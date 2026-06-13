package billing

import "time"

type Subscription struct {
	ID                     string
	OrganizationID         string
	Provider               string
	ProviderCustomerID     *string
	ProviderSubscriptionID *string
	Plan                   string
	Status                 string
	CurrentPeriodEnd       *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}
