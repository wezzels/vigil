-- TimescaleDB Setup for VIGIL
-- Run after extensions are enabled

-- Continuous aggregates for track history
CREATE MATERIALIZED VIEW track_history_minute
WITH (timescaledb.continuous) AS
SELECT 
    track_id,
    time_bucket('1 minute', time) AS bucket,
    AVG(latitude) AS avg_latitude,
    AVG(longitude) AS avg_longitude,
    AVG(altitude) AS avg_altitude,
    AVG(velocity_x) AS avg_velocity_x,
    AVG(velocity_y) AS avg_velocity_y,
    AVG(velocity_z) AS avg_velocity_z,
    COUNT(*) AS sample_count,
    MIN(latitude) AS min_latitude,
    MAX(latitude) AS max_latitude,
    MIN(longitude) AS min_longitude,
    MAX(longitude) AS max_longitude
FROM track_history
GROUP BY track_id, time_bucket('1 minute', time)
WITH DATA;

-- Create indexes on continuous aggregates
CREATE INDEX idx_track_minute_bucket ON track_history_minute (bucket DESC);
CREATE INDEX idx_track_minute_track ON track_history_minute (track_id);

-- Hour aggregate
CREATE MATERIALIZED VIEW track_history_hour
WITH (timescaledb.continuous) AS
SELECT 
    track_id,
    time_bucket('1 hour', time) AS bucket,
    AVG(latitude) AS avg_latitude,
    AVG(longitude) AS avg_longitude,
    AVG(altitude) AS avg_altitude,
    AVG(velocity_x) AS avg_velocity_x,
    AVG(velocity_y) AS avg_velocity_y,
    AVG(velocity_z) AS avg_velocity_z,
    COUNT(*) AS sample_count,
    MIN(latitude) AS min_latitude,
    MAX(latitude) AS max_latitude,
    MIN(longitude) AS min_longitude,
    MAX(longitude) AS max_longitude
FROM track_history
GROUP BY track_id, time_bucket('1 hour', time)
WITH DATA;

CREATE INDEX idx_track_hour_bucket ON track_history_hour (bucket DESC);
CREATE INDEX idx_track_hour_track ON track_history_hour (track_id);

-- Day aggregate
CREATE MATERIALIZED VIEW track_history_day
WITH (timescaledb.continuous) AS
SELECT 
    track_id,
    time_bucket('1 day', time) AS bucket,
    AVG(latitude) AS avg_latitude,
    AVG(longitude) AS avg_longitude,
    AVG(altitude) AS avg_altitude,
    AVG(velocity_x) AS avg_velocity_x,
    AVG(velocity_y) AS avg_velocity_y,
    AVG(velocity_z) AS avg_velocity_z,
    COUNT(*) AS sample_count,
    MIN(latitude) AS min_latitude,
    MAX(latitude) AS max_latitude,
    MIN(longitude) AS min_longitude,
    MAX(longitude) AS max_longitude
FROM track_history
GROUP BY track_id, time_bucket('1 day', time)
WITH DATA;

CREATE INDEX idx_track_day_bucket ON track_history_day (bucket DESC);
CREATE INDEX idx_track_day_track ON track_history_day (track_id);

-- Retention policies
SELECT add_retention_policy('track_history', INTERVAL '90 days');
SELECT add_retention_policy('track_history_minute', INTERVAL '7 days');
SELECT add_retention_policy('track_history_hour', INTERVAL '30 days');
SELECT add_retention_policy('track_history_day', INTERVAL '365 days');

-- Refresh policies
SELECT add_continuous_aggregate_policy('track_history_minute',
    start_offset => INTERVAL '2 minutes',
    end_offset => INTERVAL '1 minute',
    schedule_interval => INTERVAL '1 minute');

SELECT add_continuous_aggregate_policy('track_history_hour',
    start_offset => INTERVAL '2 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');

SELECT add_continuous_aggregate_policy('track_history_day',
    start_offset => INTERVAL '2 days',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day');

-- Compression policies
ALTER TABLE track_history SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'track_id'
);

SELECT add_compression_policy('track_history', INTERVAL '7 days');

-- Alert history hypertable
CREATE TABLE alert_history (
    time            TIMESTAMP WITH TIME ZONE NOT NULL,
    alert_id        UUID NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
    status          alert_status NOT NULL,
    priority        alert_priority NOT NULL,
    escalation_level escalation_level NOT NULL,
    recipient       VARCHAR(200),
    message         TEXT,
    
    PRIMARY KEY (time, alert_id)
);

SELECT create_hypertable('alert_history', 'time', if_not_exists => TRUE);

CREATE INDEX idx_alert_history_alert ON alert_history (alert_id);
CREATE INDEX idx_alert_history_time ON alert_history (time DESC);

-- Event log hypertable
CREATE TABLE event_log (
    time            TIMESTAMP WITH TIME ZONE NOT NULL,
    event_type      VARCHAR(100) NOT NULL,
    event_source    VARCHAR(100) NOT NULL,
    track_id        UUID REFERENCES tracks(id) ON DELETE SET NULL,
    alert_id        UUID REFERENCES alerts(id) ON DELETE SET NULL,
    data            JSONB NOT NULL DEFAULT '{}',
    severity        VARCHAR(20) DEFAULT 'info',
    
    PRIMARY KEY (time, event_type, event_source)
);

SELECT create_hypertable('event_log', 'time', if_not_exists => TRUE);

CREATE INDEX idx_event_log_type ON event_log (event_type, time DESC);
CREATE INDEX idx_event_log_track ON event_log (track_id, time DESC);
CREATE INDEX idx_event_log_data ON event_log USING GIN (data);

-- Retention for event log
SELECT add_retention_policy('event_log', INTERVAL '365 days');

-- Compression for event log
ALTER TABLE event_log SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'event_type'
);

SELECT add_compression_policy('event_log', INTERVAL '30 days');

-- Helper functions for time-series queries
CREATE OR REPLACE FUNCTION get_track_positions(
    p_track_id UUID,
    p_start_time TIMESTAMP WITH TIME ZONE,
    p_end_time TIMESTAMP WITH TIME ZONE DEFAULT NOW()
) RETURNS TABLE (
    time TIMESTAMP WITH TIME ZONE,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    altitude DOUBLE PRECISION
) AS $$
BEGIN
    RETURN QUERY
    SELECT th.time, th.latitude, th.longitude, th.altitude
    FROM track_history th
    WHERE th.track_id = p_track_id
      AND th.time >= p_start_time
      AND th.time <= p_end_time
    ORDER BY th.time;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_track_velocity(
    p_track_id UUID,
    p_start_time TIMESTAMP WITH TIME ZONE,
    p_end_time TIMESTAMP WITH TIME ZONE DEFAULT NOW()
) RETURNS TABLE (
    time TIMESTAMP WITH TIME ZONE,
    velocity_x DOUBLE PRECISION,
    velocity_y DOUBLE PRECISION,
    velocity_z DOUBLE PRECISION
) AS $$
BEGIN
    RETURN QUERY
    SELECT th.time, th.velocity_x, th.velocity_y, th.velocity_z
    FROM track_history th
    WHERE th.track_id = p_track_id
      AND th.time >= p_start_time
      AND th.time <= p_end_time
    ORDER BY th.time;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_tracks_in_area(
    p_min_lat DOUBLE PRECISION,
    p_max_lat DOUBLE PRECISION,
    p_min_lon DOUBLE PRECISION,
    p_max_lon DOUBLE PRECISION,
    p_time_threshold INTERVAL DEFAULT INTERVAL '5 minutes'
) RETURNS TABLE (
    id UUID,
    track_number VARCHAR,
    track_id VARCHAR,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    altitude DOUBLE PRECISION,
    last_update TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT t.id, t.track_number, t.track_id, t.latitude, t.longitude, t.altitude, t.last_update
    FROM tracks t
    WHERE t.latitude >= p_min_lat
      AND t.latitude <= p_max_lat
      AND t.longitude >= p_min_lon
      AND t.longitude <= p_max_lon
      AND t.last_update >= NOW() - p_time_threshold
    ORDER BY t.last_update DESC;
END;
$$ LANGUAGE plpgsql;

-- Statistics views
CREATE VIEW track_statistics AS
SELECT 
    COUNT(*) AS total_tracks,
    COUNT(*) FILTER (WHERE identity = 'hostile') AS hostile_tracks,
    COUNT(*) FILTER (WHERE identity = 'friendly') AS friendly_tracks,
    COUNT(*) FILTER (WHERE identity = 'unknown') AS unknown_tracks,
    AVG(confidence) AS avg_confidence,
    MAX(last_update) AS last_update
FROM tracks;

CREATE VIEW alert_statistics AS
SELECT 
    COUNT(*) AS total_alerts,
    COUNT(*) FILTER (WHERE status = 'pending') AS pending_alerts,
    COUNT(*) FILTER (WHERE status = 'acknowledged') AS acknowledged_alerts,
    COUNT(*) FILTER (WHERE status = 'processing') AS processing_alerts,
    COUNT(*) FILTER (WHERE status = 'complete') AS completed_alerts,
    COUNT(*) FILTER (WHERE priority = 'critical') AS critical_alerts,
    COUNT(*) FILTER (WHERE priority = 'imminent') AS imminent_alerts
FROM alerts;

-- Comments
COMMENT ON TABLE track_history IS 'Time-series track position history (TimescaleDB hypertable)';
COMMENT ON TABLE alert_history IS 'Alert state history (TimescaleDB hypertable)';
COMMENT ON TABLE event_log IS 'System event log (TimescaleDB hypertable)';
COMMENT ON VIEW track_statistics IS 'Track statistics summary';
COMMENT ON VIEW alert_statistics IS 'Alert statistics summary';