package usage

import "time"

type UsageCounter struct {
	ID             string
	OrganizationID string
	Metric         string
	Value          int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
