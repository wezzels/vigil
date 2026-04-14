-- VIGIL Database Schema Design
-- PostgreSQL 15+

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";
CREATE EXTENSION IF NOT EXISTS "timescaledb" CASCADE;

-- Enum types
CREATE TYPE track_identity AS ENUM ('unknown', 'pending', 'friendly', 'hostile', 'neutral', 'assumed_hostile');
CREATE TYPE track_quality AS ENUM ('unknown', 'poor', 'fair', 'good', 'excellent');
CREATE TYPE alert_priority AS ENUM ('low', 'normal', 'high', 'critical', 'imminent');
CREATE TYPE alert_status AS ENUM ('pending', 'acknowledged', 'processing', 'complete', 'failed', 'cancelled');
CREATE TYPE escalation_level AS ENUM ('none', 'notify', 'alert', 'critical', 'emergency');
CREATE TYPE delivery_status AS ENUM ('pending', 'sent', 'delivered', 'acknowledged', 'failed', 'timeout');

-- Tracks table
CREATE TABLE tracks (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    track_number    VARCHAR(50) NOT NULL,
    track_id        VARCHAR(100) UNIQUE NOT NULL,
    source_system   VARCHAR(100) NOT NULL,
    
    -- Position
    latitude        DOUBLE PRECISION NOT NULL,
    longitude       DOUBLE PRECISION NOT NULL,
    altitude        DOUBLE PRECISION NOT NULL DEFAULT 0,
    geom            GEOMETRY(POINT, 4326),
    
    -- Velocity
    velocity_x      DOUBLE PRECISION,
    velocity_y      DOUBLE PRECISION,
    velocity_z      DOUBLE PRECISION,
    
    -- Track properties
    identity        track_identity NOT NULL DEFAULT 'unknown',
    quality         track_quality NOT NULL DEFAULT 'unknown',
    confidence      DOUBLE PRECISION DEFAULT 0.0 CHECK (confidence >= 0 AND confidence <= 1),
    
    -- Metadata
    track_type      VARCHAR(50),
    force_id        VARCHAR(50),
    environment     VARCHAR(20),
    
    -- Timestamps
    first_detect    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_update     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Indexes
    CONSTRAINT valid_coordinates CHECK (
        latitude >= -90 AND latitude <= 90 AND
        longitude >= -180 AND longitude <= 180
    )
);

-- Create spatial index
CREATE INDEX idx_tracks_geom ON tracks USING GIST (geom);
CREATE INDEX idx_tracks_track_number ON tracks (track_number);
CREATE INDEX idx_tracks_track_id ON tracks (track_id);
CREATE INDEX idx_tracks_source ON tracks (source_system);
CREATE INDEX idx_tracks_identity ON tracks (identity);
CREATE INDEX idx_tracks_last_update ON tracks (last_update DESC);

-- Track history (TimescaleDB hypertable)
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

-- Convert to hypertable
SELECT create_hypertable('track_history', 'time', if_not_exists => TRUE);

-- Alerts table
CREATE TABLE alerts (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    alert_id        VARCHAR(100) UNIQUE NOT NULL,
    alert_type      VARCHAR(50) NOT NULL,
    priority        alert_priority NOT NULL DEFAULT 'normal',
    status          alert_status NOT NULL DEFAULT 'pending',
    
    -- Track reference
    track_id        UUID REFERENCES tracks(id) ON DELETE SET NULL,
    track_number    VARCHAR(50),
    
    -- Alert content
    message         TEXT,
    source_system   VARCHAR(100) NOT NULL,
    
    -- Escalation
    escalation_level escalation_level NOT NULL DEFAULT 'none',
    escalation_path TEXT[],
    
    -- Timing
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMP WITH TIME ZONE,
    acknowledged_at  TIMESTAMP WITH TIME ZONE,
    completed_at    TIMESTAMP WITH TIME ZONE,
    
    -- Acknowledgment
    acknowledged_by  VARCHAR(100),
    
    -- Correlation
    correlation_id  UUID,
    parent_alert_id UUID REFERENCES alerts(id) ON DELETE SET NULL
);

CREATE INDEX idx_alerts_alert_id ON alerts (alert_id);
CREATE INDEX idx_alerts_status ON alerts (status);
CREATE INDEX idx_alerts_priority ON alerts (priority);
CREATE INDEX idx_alerts_created_at ON alerts (created_at DESC);
CREATE INDEX idx_alerts_track_id ON alerts (track_id);
CREATE INDEX idx_alerts_source ON alerts (source_system);

-- Alert recipients (delivery tracking)
CREATE TABLE alert_recipients (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    alert_id        UUID NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
    recipient       VARCHAR(200) NOT NULL,
    status          delivery_status NOT NULL DEFAULT 'pending',
    attempts        INTEGER NOT NULL DEFAULT 0,
    max_attempts    INTEGER NOT NULL DEFAULT 3,
    first_attempt   TIMESTAMP WITH TIME ZONE,
    last_attempt    TIMESTAMP WITH TIME ZONE,
    delivered_at    TIMESTAMP WITH TIME ZONE,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    last_error      TEXT,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    UNIQUE (alert_id, recipient)
);

CREATE INDEX idx_recipients_alert ON alert_recipients (alert_id);
CREATE INDEX idx_recipients_status ON alert_recipients (status);

-- Events table
CREATE TABLE events (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type      VARCHAR(100) NOT NULL,
    event_source    VARCHAR(100) NOT NULL,
    event_time      TIMESTAMP WITH TIME ZONE NOT NULL,
    
    -- References
    track_id        UUID REFERENCES tracks(id) ON DELETE SET NULL,
    alert_id        UUID REFERENCES alerts(id) ON DELETE SET NULL,
    
    -- Event data (JSON)
    data            JSONB NOT NULL DEFAULT '{}',
    
    -- Metadata
    severity        VARCHAR(20) DEFAULT 'info',
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Indexes
    CONSTRAINT valid_severity CHECK (severity IN ('debug', 'info', 'warning', 'error', 'critical'))
);

CREATE INDEX idx_events_type ON events (event_type);
CREATE INDEX idx_events_time ON events (event_time DESC);
CREATE INDEX idx_events_track ON events (track_id);
CREATE INDEX idx_events_data ON events USING GIN (data);

-- Entities table (persistent entities)
CREATE TABLE entities (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_id       VARCHAR(100) UNIQUE NOT NULL,
    entity_type     VARCHAR(50) NOT NULL,
    name            VARCHAR(200),
    
    -- Track reference
    current_track_id UUID REFERENCES tracks(id) ON DELETE SET NULL,
    
    -- Properties
    properties      JSONB DEFAULT '{}',
    
    -- Metadata
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_entities_entity_id ON entities (entity_id);
CREATE INDEX idx_entities_type ON entities (entity_type);
CREATE INDEX idx_entities_track ON entities (current_track_id);

-- Correlations table
CREATE TABLE correlations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    primary_track_id UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    secondary_track_id UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    correlation_score DOUBLE PRECISION NOT NULL CHECK (correlation_score >= 0 AND correlation_score <= 1),
    correlation_type VARCHAR(50) NOT NULL,
    source_system   VARCHAR(100) NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    UNIQUE (primary_track_id, secondary_track_id)
);

CREATE INDEX idx_correlations_primary ON correlations (primary_track_id);
CREATE INDEX idx_correlations_secondary ON correlations (secondary_track_id);
CREATE INDEX idx_correlations_score ON correlations (correlation_score DESC);

-- System configuration
CREATE TABLE system_config (
    key             VARCHAR(100) PRIMARY KEY,
    value           TEXT NOT NULL,
    description     TEXT,
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_by      VARCHAR(100)
);

-- Audit log
CREATE TABLE audit_log (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    table_name      VARCHAR(100) NOT NULL,
    operation       VARCHAR(20) NOT NULL CHECK (operation IN ('INSERT', 'UPDATE', 'DELETE')),
    record_id       UUID,
    old_values      JSONB,
    new_values      JSONB,
    changed_by      VARCHAR(100),
    changed_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_table ON audit_log (table_name);
CREATE INDEX idx_audit_time ON audit_log (changed_at DESC);
CREATE INDEX idx_audit_record ON audit_log (record_id);

-- Create trigger for updating updated_at
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_tracks_updated_at
    BEFORE UPDATE ON tracks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_alerts_updated_at
    BEFORE UPDATE ON alerts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_alert_recipients_updated_at
    BEFORE UPDATE ON alert_recipients
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_entities_updated_at
    BEFORE UPDATE ON entities
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Create trigger for spatial geometry
CREATE OR REPLACE FUNCTION update_track_geom()
RETURNS TRIGGER AS $$
BEGIN
    NEW.geom = ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_tracks_geom
    BEFORE INSERT OR UPDATE ON tracks
    FOR EACH ROW EXECUTE FUNCTION update_track_geom();

-- Insert default configuration
INSERT INTO system_config (key, value, description) VALUES
    ('track_retention_days', '30', 'Days to retain track history'),
    ('alert_retention_days', '90', 'Days to retain alerts'),
    ('event_retention_days', '365', 'Days to retain events'),
    ('correlation_threshold', '0.7', 'Minimum correlation score for auto-correlation'),
    ('max_delivery_attempts', '3', 'Maximum delivery attempts for alerts'),
    ('escalation.notify_to_alert_minutes', '5', 'Minutes before escalating to ALERT'),
    ('escalation.alert_to_critical_minutes', '10', 'Minutes before escalating to CRITICAL'),
    ('escalation.critical_to_emergency_minutes', '15', 'Minutes before escalating to EMERGENCY');

-- Views for common queries
CREATE VIEW active_tracks AS
SELECT * FROM tracks
WHERE last_update > NOW() - INTERVAL '5 minutes'
ORDER BY last_update DESC;

CREATE VIEW active_alerts AS
SELECT * FROM alerts
WHERE status IN ('pending', 'acknowledged', 'processing')
ORDER BY priority DESC, created_at ASC;

CREATE VIEW pending_deliveries AS
SELECT ar.*, a.alert_type, a.priority, a.message
FROM alert_recipients ar
JOIN alerts a ON ar.alert_id = a.id
WHERE ar.status IN ('pending', 'sent')
ORDER BY a.priority DESC, ar.created_at ASC;

-- Functions for common operations
CREATE OR REPLACE FUNCTION create_alert(
    p_alert_type VARCHAR,
    p_priority alert_priority,
    p_track_id UUID DEFAULT NULL,
    p_track_number VARCHAR DEFAULT NULL,
    p_message TEXT DEFAULT NULL,
    p_source_system VARCHAR,
    p_expires_at TIMESTAMP WITH TIME ZONE DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    v_alert_id UUID;
BEGIN
    INSERT INTO alerts (
        alert_id, alert_type, priority, track_id, track_number,
        message, source_system, expires_at
    ) VALUES (
        'ALERT-' || to_char(NOW(), 'YYYYMMDDHH24MISS') || '-' || substr(md5(random()::text), 1, 6),
        p_alert_type, p_priority, p_track_id, p_track_number,
        p_message, p_source_system, p_expires_at
    ) RETURNING id INTO v_alert_id;
    
    RETURN v_alert_id;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION acknowledge_alert(
    p_alert_id UUID,
    p_acknowledged_by VARCHAR
) RETURNS VOID AS $$
BEGIN
    UPDATE alerts
    SET status = 'acknowledged',
        acknowledged_at = NOW(),
        acknowledged_by = p_acknowledged_by,
        updated_at = NOW()
    WHERE id = p_alert_id;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION complete_alert(
    p_alert_id UUID
) RETURNS VOID AS $$
BEGIN
    UPDATE alerts
    SET status = 'complete',
        completed_at = NOW(),
        updated_at = NOW()
    WHERE id = p_alert_id;
END;
$$ LANGUAGE plpgsql;

-- Comments
COMMENT ON TABLE tracks IS 'Active track data with spatial indexing';
COMMENT ON TABLE track_history IS 'Time-series track position history';
COMMENT ON TABLE alerts IS 'Alert events with escalation tracking';
COMMENT ON TABLE alert_recipients IS 'Delivery tracking per recipient';
COMMENT ON TABLE events IS 'System and security event log';
COMMENT ON TABLE entities IS 'Persistent entity registry';
COMMENT ON TABLE correlations IS 'Track correlation relationships';
COMMENT ON TABLE system_config IS 'System configuration key-value store';
COMMENT ON TABLE audit_log IS 'Database audit trail';