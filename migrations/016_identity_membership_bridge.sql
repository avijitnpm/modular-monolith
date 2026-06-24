-- Step 0: Fail if orphaned users exist (no matching identity)
DO $$
DECLARE
    orphan_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO orphan_count
    FROM users u
    WHERE NOT EXISTS (
        SELECT 1 FROM identities i WHERE i.zitadel_user_id = u.zitadel_user_id
    );
    IF orphan_count > 0 THEN
        RAISE EXCEPTION 'Cannot migrate: % users have no matching identity record', orphan_count;
    END IF;
END $$;

-- Step 1: Add nullable identity_id column
ALTER TABLE users ADD COLUMN identity_id UUID NULL;

-- Step 2: Add FK constraint
ALTER TABLE users
ADD CONSTRAINT fk_users_identity_id
FOREIGN KEY (identity_id) REFERENCES identities(id) ON DELETE RESTRICT;

-- Step 3: Backfill identity_id from identities via zitadel_user_id
UPDATE users
SET identity_id = i.id
FROM identities i
WHERE users.zitadel_user_id = i.zitadel_user_id;

-- Step 4: Set NOT NULL
ALTER TABLE users ALTER COLUMN identity_id SET NOT NULL;

-- Step 5: Create index
CREATE INDEX idx_users_identity_id ON users(identity_id);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_users_identity_id;
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_identity_id;
ALTER TABLE users DROP COLUMN IF EXISTS identity_id;
