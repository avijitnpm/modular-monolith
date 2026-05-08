-- Write your migrate up statements here

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    zitadel_org_id TEXT NOT NULL UNIQUE,

    name TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

---- create above / drop below ----

DROP TABLE IF EXISTS organizations;
