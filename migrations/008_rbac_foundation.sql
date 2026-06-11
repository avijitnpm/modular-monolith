CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, id),
    UNIQUE (organization_id, name)
);

CREATE TABLE role_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id TEXT NOT NULL,
    role_id UUID NOT NULL,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (role_id, permission_id),
    FOREIGN KEY (organization_id, role_id)
        REFERENCES roles(organization_id, id)
        ON DELETE CASCADE
);

CREATE TABLE user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, user_id, role_id),
    FOREIGN KEY (organization_id, role_id)
        REFERENCES roles(organization_id, id)
        ON DELETE CASCADE
);

CREATE INDEX idx_roles_org
ON roles(organization_id);

CREATE INDEX idx_role_permissions_org_role
ON role_permissions(organization_id, role_id);

CREATE INDEX idx_role_permissions_permission
ON role_permissions(permission_id);

CREATE INDEX idx_user_roles_org_user
ON user_roles(organization_id, user_id);

CREATE INDEX idx_user_roles_org_role
ON user_roles(organization_id, role_id);

ALTER TABLE roles
ENABLE ROW LEVEL SECURITY;

ALTER TABLE role_permissions
ENABLE ROW LEVEL SECURITY;

ALTER TABLE user_roles
ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_roles
ON roles
USING (
    organization_id =
    current_setting(
        'app.current_organization_id',
        true
    )
)
WITH CHECK (
    organization_id =
    current_setting(
        'app.current_organization_id',
        true
    )
);

CREATE POLICY tenant_isolation_role_permissions
ON role_permissions
USING (
    organization_id =
    current_setting(
        'app.current_organization_id',
        true
    )
)
WITH CHECK (
    organization_id =
    current_setting(
        'app.current_organization_id',
        true
    )
);

CREATE POLICY tenant_isolation_user_roles
ON user_roles
USING (
    organization_id =
    current_setting(
        'app.current_organization_id',
        true
    )
)
WITH CHECK (
    organization_id =
    current_setting(
        'app.current_organization_id',
        true
    )
);

INSERT INTO permissions (name)
VALUES
    ('users.read'),
    ('users.write'),
    ('billing.read'),
    ('billing.write'),
    ('audit.read'),
    ('settings.read'),
    ('settings.write')
ON CONFLICT (name) DO NOTHING;

INSERT INTO roles (organization_id, name)
SELECT DISTINCT o.organization_id, r.name
FROM organizations o
CROSS JOIN (
    VALUES
        ('owner'),
        ('admin'),
        ('member'),
        ('viewer')
) AS r(name)
WHERE o.organization_id <> ''
ON CONFLICT (organization_id, name) DO NOTHING;

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
WHERE r.name IN ('owner', 'admin', 'member', 'viewer')
ON CONFLICT (role_id, permission_id) DO NOTHING;

---- create above / drop below ----

DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS permissions;
