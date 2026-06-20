-- Enable Row Level Security on audit_logs

ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_audit_logs
ON audit_logs
USING (
    organization_id::text =
    current_setting(
        'app.current_organization_id',
        true
    )
);

---- create above / drop below ----

DROP POLICY IF EXISTS tenant_isolation_audit_logs ON audit_logs;
ALTER TABLE audit_logs DISABLE ROW LEVEL SECURITY;
