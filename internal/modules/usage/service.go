package usage

import "context"

type Store interface {
	GetUsage(ctx context.Context, organizationID string, metric string) (*UsageCounter, error)
	IncrementUsage(ctx context.Context, organizationID string, metric string, amount int64) (*UsageCounter, error)
	SetUsage(ctx context.Context, organizationID string, metric string, value int64) (*UsageCounter, error)
	ListUsage(ctx context.Context, organizationID string) ([]UsageCounter, error)
}

type Service struct {
	Repository Store
}

func NewService(repository Store) *Service {
	return &Service{Repository: repository}
}

func (s *Service) IncrementMetric(ctx context.Context, organizationID string, metric string, amount int64) (*UsageCounter, error) {
	return s.Repository.IncrementUsage(ctx, organizationID, metric, amount)
}

func (s *Service) IncrementUsers(ctx context.Context, organizationID string) (*UsageCounter, error) {
	return s.Repository.IncrementUsage(ctx, organizationID, "users", 1)
}

func (s *Service) IncrementDocuments(ctx context.Context, organizationID string) (*UsageCounter, error) {
	return s.Repository.IncrementUsage(ctx, organizationID, "documents", 1)
}

func (s *Service) IncrementAPIRequests(ctx context.Context, organizationID string) (*UsageCounter, error) {
	return s.Repository.IncrementUsage(ctx, organizationID, "api_requests", 1)
}

func (s *Service) IncrementStorage(ctx context.Context, organizationID string, bytes int64) (*UsageCounter, error) {
	return s.Repository.IncrementUsage(ctx, organizationID, "storage", bytes)
}
