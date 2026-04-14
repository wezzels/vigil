-- Migration: 004_events_entities
-- Description: Rollback events, entities, and correlations

-- Drop triggers
DROP TRIGGER IF EXISTS update_entities_updated_at ON entities;

-- Drop indexes
DROP INDEX IF EXISTS idx_correlations_primary;
DROP INDEX IF EXISTS idx_correlations_secondary;
DROP INDEX IF EXISTS idx_correlations_score;
DROP INDEX IF EXISTS idx_entities_entity_id;
DROP INDEX IF EXISTS idx_entities_type;
DROP INDEX IF EXISTS idx_entities_track;
DROP INDEX IF EXISTS idx_events_type;
DROP INDEX IF EXISTS idx_events_time;
DROP INDEX IF EXISTS idx_events_track;
DROP INDEX IF EXISTS idx_events_data;

-- Drop tables
DROP TABLE IF EXISTS correlations;
DROP TABLE IF EXISTS entities;
DROP TABLE IF EXISTS events;