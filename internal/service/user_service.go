package service

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/avijitnpm/modular-monolith/internal/repository"
	appErrors "github.com/avijitnpm/modular-monolith/pkg/errors"
)

func (s *Service) RegisterUser(
	ctx context.Context,
	organizationID string,
	zitadelUserID string,
	email string,
) (*repository.User, error) {

	return s.Repository.CreateUser(
		ctx,
		organizationID,
		zitadelUserID,
		email,
	)
}

func (s *Service) ProvisionAuthenticatedUser(
	ctx context.Context,
	zitadelUserID string,
	email string,
	organizationID string,
) error {

	zitadelUserID = strings.TrimSpace(zitadelUserID)

	if zitadelUserID == "" {
		return errors.New("zitadel user id is required")
	}

	_, err := s.Repository.FindUserByZitadelUserID(
		ctx,
		organizationID,
		zitadelUserID,
	)

	if err == nil {
		return nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	email = strings.TrimSpace(email)

	email = provisionedUserEmail(
		zitadelUserID,
		email,
	)

	_, err = s.Repository.CreateUser(
		ctx,
		organizationID,
		zitadelUserID,
		email,
	)

	if errors.Is(err, appErrors.ErrUserAlreadyExists) {
		_, err = s.Repository.FindUserByZitadelUserID(
			ctx,
			organizationID,
			zitadelUserID,
		)

		return err
	}

	return err
}

func provisionedUserEmail(
	zitadelUserID string,
	email string,
) string {

	email = strings.TrimSpace(email)

	if email != "" {
		return email
	}

	return zitadelUserID + "@zitadel.local"
}
