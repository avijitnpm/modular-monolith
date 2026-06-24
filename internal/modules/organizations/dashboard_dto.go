package organizations

import "time"

type DashboardOrgInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DashboardSubscription struct {
	Plan             string     `json:"plan"`
	Status           string     `json:"status"`
	CurrentPeriodEnd *time.Time `json:"current_period_end"`
}

type DashboardUsage struct {
	Users       int64 `json:"users"`
	Documents   int64 `json:"documents"`
	APIRequests int64 `json:"api_requests"`
	Storage     int64 `json:"storage"`
}

type DashboardEntitlement struct {
	Metric    string `json:"metric"`
	Used      int64  `json:"used"`
	Limit     int64  `json:"limit"`
	Remaining int64  `json:"remaining"`
	Allowed   bool   `json:"allowed"`
}

type DashboardResponse struct {
	Organization DashboardOrgInfo       `json:"organization"`
	Subscription *DashboardSubscription `json:"subscription"`
	Usage        DashboardUsage         `json:"usage"`
	Entitlements []DashboardEntitlement `json:"entitlements"`
}

type SummaryResponse struct {
	OrganizationID   string `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
	Plan             string `json:"plan"`
	Status           string `json:"status"`
}

type UsageSummaryResponse struct {
	Users       int64 `json:"users"`
	Documents   int64 `json:"documents"`
	APIRequests int64 `json:"api_requests"`
	Storage     int64 `json:"storage"`
}
