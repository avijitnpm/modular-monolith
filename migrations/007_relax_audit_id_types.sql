ALTER TABLE audit_logs
ALTER COLUMN organization_id TYPE TEXT
USING organization_id::text;

ALTER TABLE audit_logs
ALTER COLUMN user_id TYPE TEXT
USING user_id::text;