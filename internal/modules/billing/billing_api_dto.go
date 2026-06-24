package billing

import "time"

type SubscriptionDetailResponse struct {
	Plan             string     `json:"plan"`
	Status           string     `json:"status"`
	Provider         string     `json:"provider"`
	CurrentPeriodEnd *time.Time `json:"current_period_end"`
}

type UsageMetricsResponse struct {
	Users       int64 `json:"users"`
	Documents   int64 `json:"documents"`
	APIRequests int64 `json:"api_requests"`
	Storage     int64 `json:"storage"`
}

type EntitlementItem struct {
	Metric    string `json:"metric"`
	Used      int64  `json:"used"`
	Limit     int64  `json:"limit"`
	Remaining int64  `json:"remaining"`
	Allowed   bool   `json:"allowed"`
}

type EntitlementsResponse struct {
	Entitlements []EntitlementItem `json:"entitlements"`
}
