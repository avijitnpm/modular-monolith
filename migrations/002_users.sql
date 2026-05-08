-- Write your migrate up statements here
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    zitadel_user_id TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
---- create above / drop below ----

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
DROP TABLE IF EXISTS users;