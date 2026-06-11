package rbac

import (
	"encoding/json"
	"time"
)

type permissionRow struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func decodePermissions(
	raw []byte,
) ([]Permission, error) {

	var rows []permissionRow

	err := json.Unmarshal(
		raw,
		&rows,
	)

	if err != nil {
		return nil, err
	}

	permissions := make(
		[]Permission,
		0,
		len(rows),
	)

	for _, row := range rows {
		permissions = append(
			permissions,
			Permission{
				ID:        row.ID,
				Name:      row.Name,
				CreatedAt: row.CreatedAt,
			},
		)
	}

	return permissions, nil
}

func uniqueStrings(
	values []string,
) []string {

	seen := map[string]struct{}{}
	unique := []string{}

	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}

		unique = append(
			unique,
			value,
		)
	}

	return unique
}
