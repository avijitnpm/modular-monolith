UPDATE organizations
SET organization_id = zitadel_org_id
WHERE organization_id = ''
	AND zitadel_org_id <> '';

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

DELETE FROM role_permissions rp
USING roles r
WHERE rp.role_id = r.id
	AND r.name IN ('owner', 'admin', 'member', 'viewer');

DELETE FROM roles
WHERE name IN ('owner', 'admin', 'member', 'viewer');
