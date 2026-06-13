CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_customer_id TEXT,
    provider_subscription_id TEXT,
    plan TEXT NOT NULL,
    status TEXT NOT NULL,
    current_period_end TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_subscriptions_org
ON subscriptions(organization_id);

CREATE UNIQUE INDEX idx_subscriptions_provider_customer
ON subscriptions(provider, provider_customer_id)
WHERE provider_customer_id IS NOT NULL;

CREATE UNIQUE INDEX idx_subscriptions_provider_subscription
ON subscriptions(provider, provider_subscription_id)
WHERE provider_subscription_id IS NOT NULL;

ALTER TABLE subscriptions
ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_subscriptions
ON subscriptions
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

DROP TABLE IF EXISTS subscriptions;
