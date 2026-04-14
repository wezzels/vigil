-- Migration: 003_track_history
-- Description: Rollback track history

-- Drop retention policy
SELECT remove_retention_policy('track_history', if_exists => TRUE);

-- Drop continuous aggregate
DROP MATERIALIZED VIEW IF EXISTS track_history_recent CASCADE;

-- Drop indexes
DROP INDEX IF EXISTS idx_track_history_track;
DROP INDEX IF EXISTS idx_track_history_time;

-- Drop hypertable
DROP TABLE IF EXISTS track_history CASCADE;