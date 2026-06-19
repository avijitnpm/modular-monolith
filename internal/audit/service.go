package audit

import (
	"context"

	"github.com/avijitnpm/modular-monolith/internal/repository"
)

type Service struct {
	Repository *repository.Repository
}

func NewService(
	repository *repository.Repository,
) *Service {

	return &Service{
		Repository: repository,
	}
}

func (s *Service) Log(
	ctx context.Context,
	event *Event,
) error {

	return s.Repository.CreateAuditLog(
		ctx,
		event.OrganizationID,
		event.UserID,
		event.Action,
		event.EntityType,
		event.EntityID,
		event.Metadata,
	)
}

func (s *Service) List(
	ctx context.Context,
	organizationID string,
	limit int,
	offset int,
) ([]repository.AuditLog, error) {

	return s.Repository.ListAuditLogs(
		ctx,
		organizationID,
		limit,
		offset,
	)
}
