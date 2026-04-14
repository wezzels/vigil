-- Create indexes for optimal query performance
-- Migration: 005_indexes

-- Track ID index
CREATE INDEX IF NOT EXISTS idx_tracks_track_id ON tracks(track_id);

-- Timestamp indexes
CREATE INDEX IF NOT EXISTS idx_tracks_created_at ON tracks(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tracks_updated_at ON tracks(updated_at DESC);

-- Source index for filtering
CREATE INDEX IF NOT EXISTS idx_tracks_source ON tracks(source);

-- Status index
CREATE INDEX IF NOT EXISTS idx_tracks_status ON tracks(status);

-- PostGIS spatial index (if PostGIS extension is available)
-- Note: Requires CREATE EXTENSION postgis; to be run first
CREATE INDEX IF NOT EXISTS idx_tracks_location ON tracks USING GIST (
    ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)
);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_tracks_source_created ON tracks(source, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tracks_source_status ON tracks(source, status);

-- Alerts indexes
CREATE INDEX IF NOT EXISTS idx_alerts_track_id ON alerts(track_id);
CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
CREATE INDEX IF NOT EXISTS idx_alerts_priority ON alerts(priority);

-- Events indexes
CREATE INDEX IF NOT EXISTS idx_events_track_id ON events(track_id);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp DESC);

-- TimescaleDB hypertable indexes (if using TimescaleDB)
-- These are created automatically when creating hypertables
-- but can be added explicitly for optimization
CREATE INDEX IF NOT EXISTS idx_events_track_time ON events(track_id, timestamp DESC);

-- Partial indexes for active tracks
CREATE INDEX IF NOT EXISTS idx_tracks_active ON tracks(track_id, updated_at DESC) 
WHERE status = 'active';

-- Partial index for pending alerts
CREATE INDEX IF NOT EXISTS idx_alerts_pending ON alerts(track_id, created_at DESC) 
WHERE status = 'pending';

-- BRIN index for large time-series tables (better for bulk inserts)
CREATE INDEX IF NOT EXISTS idx_events_timestamp_brin ON events USING BRIN (timestamp);

-- Comments for documentation
COMMENT ON INDEX idx_tracks_track_id IS 'Primary lookup index for tracks';
COMMENT ON INDEX idx_tracks_created_at IS 'Time-based queries for tracks';
COMMENT ON INDEX idx_tracks_location IS 'Spatial queries using PostGIS';
COMMENT ON INDEX idx_tracks_source_created IS 'Optimized queries filtering by source with time ordering';
COMMENT ON INDEX idx_tracks_active IS 'Partial index for active tracks only';
COMMENT ON INDEX idx_alerts_pending IS 'Partial index for pending alerts only';