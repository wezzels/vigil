-- Migration: 001_tracks
-- Description: Rollback tracks table

-- Drop triggers
DROP TRIGGER IF EXISTS update_tracks_geom ON tracks;
DROP TRIGGER IF EXISTS update_tracks_updated_at ON tracks;

-- Drop indexes
DROP INDEX IF EXISTS idx_tracks_geom;
DROP INDEX IF EXISTS idx_tracks_track_number;
DROP INDEX IF EXISTS idx_tracks_track_id;
DROP INDEX IF EXISTS idx_tracks_source;
DROP INDEX IF EXISTS idx_tracks_identity;
DROP INDEX IF EXISTS idx_tracks_last_update;

-- Drop table
DROP TABLE IF EXISTS tracks;