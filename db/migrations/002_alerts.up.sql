-- Migration: 002_alerts
-- Description: Alerts and alert_recipients tables

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

-- Create indexes
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

-- Create indexes
CREATE INDEX idx_recipients_alert ON alert_recipients (alert_id);
CREATE INDEX idx_recipients_status ON alert_recipients (status);

-- Create triggers
CREATE TRIGGER update_alerts_updated_at
    BEFORE UPDATE ON alerts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_alert_recipients_updated_at
    BEFORE UPDATE ON alert_recipients
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();