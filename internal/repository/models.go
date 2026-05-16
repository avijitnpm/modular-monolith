package repository

import "time"

type User struct {
	ID string

	ZitadelUserID string
	Email         string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type Organization struct {
	ID string

	ZitadelOrgID string
	Name         string

	CreatedAt time.Time
}
