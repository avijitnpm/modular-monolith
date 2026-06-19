package service

import (
	"context"

	"github.com/avijitnpm/modular-monolith/internal/audit"
	"github.com/avijitnpm/modular-monolith/internal/repository"
)

type AuditLogger interface {
	Log(ctx context.Context, event *audit.Event) error
}

type Service struct {
	Repository *repository.Repository
	Audit      AuditLogger
}

func New(
	repo *repository.Repository,
	auditLogger AuditLogger,
) *Service {

	return &Service{
		Repository: repo,
		Audit:      auditLogger,
	}
}
