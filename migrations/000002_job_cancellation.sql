BEGIN;
ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_status_check;
ALTER TABLE jobs ADD CONSTRAINT jobs_status_check CHECK (status IN ('pending','running','succeeded','dead','cancelled'));
CREATE INDEX IF NOT EXISTS jobs_created_id_idx ON jobs(created_at DESC,id DESC);
COMMIT;
