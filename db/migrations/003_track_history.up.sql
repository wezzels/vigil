-- Migration: 003_track_history
-- Description: Track history hypertable for TimescaleDB

-- Track history table
CREATE TABLE track_history (
    time            TIMESTAMP WITH TIME ZONE NOT NULL,
    track_id        UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    latitude        DOUBLE PRECISION NOT NULL,
    longitude       DOUBLE PRECISION NOT NULL,
    altitude        DOUBLE PRECISION NOT NULL,
    velocity_x      DOUBLE PRECISION,
    velocity_y      DOUBLE PRECISION,
    velocity_z      DOUBLE PRECISION,
    identity        track_identity,
    quality         track_quality,
    confidence      DOUBLE PRECISION,
    
    PRIMARY KEY (time, track_id)
);

-- Create hypertable
SELECT create_hypertable('track_history', 'time', if_not_exists => TRUE);

-- Create indexes
CREATE INDEX idx_track_history_track ON track_history (track_id, time DESC);
CREATE INDEX idx_track_history_time ON track_history (time DESC);

-- Continuous aggregate for recent tracks
CREATE MATERIALIZED VIEW track_history_recent
WITH (timescaledb.continuous) AS
SELECT 
    track_id,
    time,
    latitude,
    longitude,
    altitude,
    velocity_x,
    velocity_y,
    velocity_z,
    identity,
    quality,
    confidence
FROM track_history
WHERE time > NOW() - INTERVAL '1 hour'
WITH DATA;

-- Refresh policy
SELECT add_continuous_aggregate_policy('track_history_recent',
    start_offset => INTERVAL '2 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '5 minutes');

-- Retention policy (keep 90 days)
SELECT add_retention_policy('track_history', INTERVAL '90 days');