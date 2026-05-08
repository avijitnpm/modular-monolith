-- Write your migrate up statements here

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    organization_id UUID NOT NULL,
    user_id UUID,

    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT,

    metadata JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_org
ON audit_logs(organization_id);

CREATE INDEX idx_audit_logs_created
ON audit_logs(created_at);

---- create above / drop below ----

DROP TABLE IF EXISTS audit_logs;
