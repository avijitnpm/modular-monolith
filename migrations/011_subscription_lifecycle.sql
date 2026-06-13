CREATE UNIQUE INDEX idx_subscriptions_organization
ON subscriptions(organization_id);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_subscriptions_organization;
