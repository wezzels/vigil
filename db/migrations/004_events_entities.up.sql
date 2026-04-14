-- Migration: 004_events_entities
-- Description: Events, entities, and correlations tables

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
    
    -- Constraints
    CONSTRAINT valid_severity CHECK (severity IN ('debug', 'info', 'warning', 'error', 'critical'))
);

-- Create indexes
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

-- Create indexes
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

-- Create indexes
CREATE INDEX idx_correlations_primary ON correlations (primary_track_id);
CREATE INDEX idx_correlations_secondary ON correlations (secondary_track_id);
CREATE INDEX idx_correlations_score ON correlations (correlation_score DESC);

-- Create triggers
CREATE TRIGGER update_entities_updated_at
    BEFORE UPDATE ON entities
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();