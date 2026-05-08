package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func New(databaseURL string) (*pgxpool.Pool, error) {

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	defer cancel()

	db, err := pgxpool.New(ctx, databaseURL)

	if err != nil {
		return nil, err
	}

	err = db.Ping(ctx)

	if err != nil {
		return nil, err
	}

	return db, nil
}
