package rbac

import (
	"context"
	"errors"

	"github.com/avijitnpm/modular-monolith/internal/database"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appErrors "github.com/avijitnpm/modular-monolith/pkg/errors"
)

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(
	db *pgxpool.Pool,
) *Repository {

	return &Repository{
		DB: db,
	}
}

func (r *Repository) ListPermissions(
	ctx context.Context,
) ([]Permission, error) {

	rows, err := r.DB.Query(
		ctx,
		`
			SELECT id, name, created_at
			FROM permissions
			ORDER BY name
		`,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	permissions := []Permission{}

	for rows.Next() {
		var permission Permission

		err = rows.Scan(
			&permission.ID,
			&permission.Name,
			&permission.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		permissions = append(
			permissions,
			permission,
		)
	}

	return permissions, rows.Err()
}

func (r *Repository) ListRoles(
	ctx context.Context,
	organizationID string,
) ([]Role, error) {

	var roles []Role

	err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		rows, err := tx.Query(
			ctx,
			`
				SELECT
					r.id,
					r.organization_id,
					r.name,
					r.created_at,
					r.updated_at,
					COALESCE(
						jsonb_agg(
							jsonb_build_object(
								'id', p.id::text,
								'name', p.name,
								'created_at', p.created_at
							)
							ORDER BY p.name
						) FILTER (WHERE p.id IS NOT NULL),
						'[]'::jsonb
					)
				FROM roles r
				LEFT JOIN role_permissions rp
					ON rp.role_id = r.id
					AND rp.organization_id = r.organization_id
				LEFT JOIN permissions p
					ON p.id = rp.permission_id
				WHERE r.organization_id = $1
				GROUP BY r.id, r.organization_id, r.name, r.created_at, r.updated_at
				ORDER BY r.name
			`,
			organizationID,
		)

		if err != nil {
			return err
		}

		defer rows.Close()

		roles = []Role{}

		for rows.Next() {
			var role Role
			var permissionsJSON []byte

			err = rows.Scan(
				&role.ID,
				&role.OrganizationID,
				&role.Name,
				&role.CreatedAt,
				&role.UpdatedAt,
				&permissionsJSON,
			)

			if err != nil {
				return err
			}

			role.Permissions, err = decodePermissions(
				permissionsJSON,
			)

			if err != nil {
				return err
			}

			roles = append(
				roles,
				role,
			)
		}

		return rows.Err()
	})

	if err != nil {
		return nil, err
	}

	return roles, nil
}

func (r *Repository) CreateRole(
	ctx context.Context,
	organizationID string,
	name string,
	permissionNames []string,
) (*Role, error) {

	var createdRole *Role

	err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		role, err := insertRole(
			ctx,
			tx,
			organizationID,
			name,
		)

		if err != nil {
			return err
		}

		err = assignRolePermissions(
			ctx,
			tx,
			organizationID,
			role.ID,
			permissionNames,
		)

		if err != nil {
			return err
		}

		createdRole, err = getRole(
			ctx,
			tx,
			organizationID,
			role.ID,
		)

		return err
	})

	if err != nil {
		return nil, err
	}

	return createdRole, nil
}

func (r *Repository) BootstrapDefaultRoles(
	ctx context.Context,
	organizationID string,
) error {

	return database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		return BootstrapDefaultRolesTx(
			ctx,
			tx,
			organizationID,
		)
	})
}

func (r *Repository) AssignRoleToUser(
	ctx context.Context,
	organizationID string,
	userID string,
	roleID string,
) (*UserRole, error) {

	assignment := &UserRole{}

	err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			`
				INSERT INTO user_roles (organization_id, user_id, role_id)
				VALUES ($1, $2, $3)
				RETURNING id, organization_id, user_id, role_id, created_at
			`,
			organizationID,
			userID,
			roleID,
		).Scan(
			&assignment.ID,
			&assignment.OrganizationID,
			&assignment.UserID,
			&assignment.RoleID,
			&assignment.CreatedAt,
		)
	})

	if err != nil {
		return nil, mapConstraintError(err)
	}

	return assignment, nil
}

func (r *Repository) RemoveRoleFromUser(
	ctx context.Context,
	organizationID string,
	userID string,
	roleID string,
) (*UserRole, error) {

	assignment := &UserRole{}

	err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			`
				DELETE FROM user_roles
				WHERE organization_id = $1
					AND user_id = $2
					AND role_id = $3
				RETURNING id, organization_id, user_id, role_id, created_at
			`,
			organizationID,
			userID,
			roleID,
		).Scan(
			&assignment.ID,
			&assignment.OrganizationID,
			&assignment.UserID,
			&assignment.RoleID,
			&assignment.CreatedAt,
		)
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserRoleNotFound
	}

	if err != nil {
		return nil, err
	}

	return assignment, nil
}

func (r *Repository) UserHasPermission(
	ctx context.Context,
	organizationID string,
	zitadelUserID string,
	permission string,
) (bool, error) {

	var exists bool

	err := database.WithTenantQuery(r.DB, ctx, organizationID, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx,
			`
				SELECT EXISTS (
					SELECT 1
					FROM users u
					JOIN user_roles ur
						ON ur.user_id = u.id
						AND ur.organization_id = u.organization_id
					JOIN role_permissions rp
						ON rp.role_id = ur.role_id
						AND rp.organization_id = ur.organization_id
					JOIN permissions p
						ON p.id = rp.permission_id
					WHERE u.organization_id = $1
						AND u.zitadel_user_id = $2
						AND p.name = $3
				)
			`,
			organizationID,
			zitadelUserID,
			permission,
		).Scan(
			&exists,
		)
	})

	return exists, err
}

func BootstrapDefaultRolesTx(
	ctx context.Context,
	tx pgx.Tx,
	organizationID string,
) error {

	if err := database.SetTenantContext(ctx, tx, organizationID); err != nil {
		return err
	}

	_, err := tx.Exec(
		ctx,
		`
			INSERT INTO roles (organization_id, name)
			VALUES
				($1, 'owner'),
				($1, 'admin'),
				($1, 'member'),
				($1, 'viewer')
			ON CONFLICT (organization_id, name) DO NOTHING
		`,
		organizationID,
	)

	if err != nil {
		return mapConstraintError(err)
	}

	_, err = tx.Exec(
		ctx,
		`
			INSERT INTO role_permissions (organization_id, role_id, permission_id)
			SELECT r.organization_id, r.id, p.id
			FROM roles r
			JOIN permissions p
				ON (
					r.name = 'owner'
					OR (r.name = 'admin' AND p.name <> 'billing.write')
					OR (r.name = 'member' AND p.name IN ('users.read', 'settings.read'))
					OR (r.name = 'viewer' AND p.name IN (
						'users.read',
						'billing.read',
						'audit.read',
						'settings.read'
					))
				)
			WHERE r.organization_id = $1
				AND r.name IN ('owner', 'admin', 'member', 'viewer')
			ON CONFLICT (role_id, permission_id) DO NOTHING
		`,
		organizationID,
	)

	if err != nil {
		return mapConstraintError(err)
	}

	return nil
}

func insertRole(
	ctx context.Context,
	tx pgx.Tx,
	organizationID string,
	name string,
) (*Role, error) {

	role := &Role{}

	err := tx.QueryRow(
		ctx,
		`
			INSERT INTO roles (organization_id, name)
			VALUES ($1, $2)
			RETURNING id, organization_id, name, created_at, updated_at
		`,
		organizationID,
		name,
	).Scan(
		&role.ID,
		&role.OrganizationID,
		&role.Name,
		&role.CreatedAt,
		&role.UpdatedAt,
	)

	if err != nil {
		return nil, mapConstraintError(err)
	}

	return role, nil
}

func assignRolePermissions(
	ctx context.Context,
	tx pgx.Tx,
	organizationID string,
	roleID string,
	permissionNames []string,
) error {

	if len(permissionNames) == 0 {
		return nil
	}

	tag, err := tx.Exec(
		ctx,
		`
			INSERT INTO role_permissions (organization_id, role_id, permission_id)
			SELECT $1, $2, p.id
			FROM permissions p
			WHERE p.name = ANY($3::text[])
			ON CONFLICT (role_id, permission_id) DO NOTHING
		`,
		organizationID,
		roleID,
		permissionNames,
	)

	if err != nil {
		return mapConstraintError(err)
	}

	if tag.RowsAffected() != int64(len(uniqueStrings(permissionNames))) {
		return ErrUnknownPermissions
	}

	return nil
}

func getRole(
	ctx context.Context,
	tx pgx.Tx,
	organizationID string,
	roleID string,
) (*Role, error) {

	role := &Role{}
	var permissionsJSON []byte

	err := tx.QueryRow(
		ctx,
		`
			SELECT
				r.id,
				r.organization_id,
				r.name,
				r.created_at,
				r.updated_at,
				COALESCE(
					jsonb_agg(
						jsonb_build_object(
							'id', p.id::text,
							'name', p.name,
							'created_at', p.created_at
						)
						ORDER BY p.name
					) FILTER (WHERE p.id IS NOT NULL),
					'[]'::jsonb
				)
			FROM roles r
			LEFT JOIN role_permissions rp
				ON rp.role_id = r.id
				AND rp.organization_id = r.organization_id
			LEFT JOIN permissions p
				ON p.id = rp.permission_id
			WHERE r.organization_id = $1
				AND r.id = $2
			GROUP BY r.id, r.organization_id, r.name, r.created_at, r.updated_at
		`,
		organizationID,
		roleID,
	).Scan(
		&role.ID,
		&role.OrganizationID,
		&role.Name,
		&role.CreatedAt,
		&role.UpdatedAt,
		&permissionsJSON,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRoleNotFound
	}

	if err != nil {
		return nil, err
	}

	role.Permissions, err = decodePermissions(
		permissionsJSON,
	)

	if err != nil {
		return nil, err
	}

	return role, nil
}

func mapConstraintError(
	err error,
) error {

	pgErr, ok := err.(*pgconn.PgError)

	if !ok {
		return err
	}

	switch pgErr.Code {
	case appErrors.PostgresUniqueViolation:
		switch pgErr.ConstraintName {
		case "roles_organization_id_name_key":
			return ErrRoleAlreadyExists
		case "user_roles_organization_id_user_id_role_id_key":
			return ErrUserRoleExists
		default:
			return err
		}
	case appErrors.PostgresForeignKeyViolation:
		return ErrRoleNotFound
	default:
		return err
	}
}
