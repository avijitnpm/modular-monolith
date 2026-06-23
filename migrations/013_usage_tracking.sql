CREATE TABLE usage_counters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id TEXT NOT NULL,
    metric TEXT NOT NULL,
    value BIGINT NOT NULL DEFAULT 0 CHECK (value >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(organization_id, metric)
);

CREATE INDEX idx_usage_counters_org
ON usage_counters(organization_id);

ALTER TABLE usage_counters
ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_usage_counters
ON usage_counters
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

---- create above / drop below ----

DROP TABLE IF EXISTS usage_counters;
