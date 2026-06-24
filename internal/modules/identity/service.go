package identity

import "context"

type Store interface {
	GetByZitadelUserID(ctx context.Context, zitadelUserID string) (*Identity, error)
	GetByEmail(ctx context.Context, email string) (*Identity, error)
	Create(ctx context.Context, zitadelUserID, email, name string) (*Identity, error)
	Update(ctx context.Context, zitadelUserID, email, name string) (*Identity, error)
}

type Service struct {
	Repository Store
}

func NewService(repository Store) *Service {
	return &Service{Repository: repository}
}

func (s *Service) FindOrCreateIdentity(ctx context.Context, zitadelUserID, email, name string) (*Identity, error) {
	existing, err := s.Repository.GetByZitadelUserID(ctx, zitadelUserID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.Email != email || existing.Name != name {
			return s.Repository.Update(ctx, zitadelUserID, email, name)
		}
		return existing, nil
	}
	return s.Repository.Create(ctx, zitadelUserID, email, name)
}

func (s *Service) GetIdentity(ctx context.Context, zitadelUserID string) (*Identity, error) {
	return s.Repository.GetByZitadelUserID(ctx, zitadelUserID)
}
