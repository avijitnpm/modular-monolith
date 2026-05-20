-- Add organization_id columns

ALTER TABLE users
ADD COLUMN organization_id TEXT NOT NULL DEFAULT '';

ALTER TABLE organizations
ADD COLUMN organization_id TEXT NOT NULL DEFAULT '';

-- Backfill temporary values

UPDATE users
SET organization_id = 'org-456';

UPDATE organizations
SET organization_id = 'org-456';

-- Enable Row Level Security

ALTER TABLE users
ENABLE ROW LEVEL SECURITY;

ALTER TABLE organizations
ENABLE ROW LEVEL SECURITY;

-- Users Policy

CREATE POLICY tenant_isolation_users
ON users
USING (
	organization_id =
	current_setting(
		'app.current_organization_id',
		true
	)
);

-- Organizations Policy

CREATE POLICY tenant_isolation_organizations
ON organizations
USING (
	organization_id =
	current_setting(
		'app.current_organization_id',
		true
	)
);