BEGIN;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS client_id varchar(64) NOT NULL DEFAULT 'default';
DROP INDEX IF EXISTS jobs_idempotency_key_unique;
CREATE UNIQUE INDEX IF NOT EXISTS jobs_client_idempotency_key_unique ON jobs(client_id,idempotency_key) WHERE idempotency_key IS NOT NULL;
COMMIT;
