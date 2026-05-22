ALTER TABLE audit_logs
ALTER COLUMN organization_id DROP NOT NULL;

ALTER TABLE audit_logs
ALTER COLUMN entity_type DROP NOT NULL;