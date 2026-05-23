package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func New(
	databaseURL string,
	tracingEnabled bool,
) (*pgxpool.Pool, error) {

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)

	defer cancel()

	cfg, err := pgxpool.ParseConfig(databaseURL)

	if err != nil {
		return nil, err
	}

	if tracingEnabled {
		cfg.ConnConfig.Tracer = NewTracer()
	}

	db, err := pgxpool.NewWithConfig(ctx, cfg)

	if err != nil {
		return nil, err
	}

	err = db.Ping(ctx)

	if err != nil {
		return nil, err
	}

	return db, nil
}
