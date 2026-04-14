-- Migration: 001_tracks
-- Description: Initial tracks table creation

-- Create tracks table
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
    
    -- Constraints
    CONSTRAINT valid_coordinates CHECK (
        latitude >= -90 AND latitude <= 90 AND
        longitude >= -180 AND longitude <= 180
    )
);

-- Create indexes
CREATE INDEX idx_tracks_geom ON tracks USING GIST (geom);
CREATE INDEX idx_tracks_track_number ON tracks (track_number);
CREATE INDEX idx_tracks_track_id ON tracks (track_id);
CREATE INDEX idx_tracks_source ON tracks (source_system);
CREATE INDEX idx_tracks_identity ON tracks (identity);
CREATE INDEX idx_tracks_last_update ON tracks (last_update DESC);

-- Create triggers
CREATE TRIGGER update_tracks_updated_at
    BEFORE UPDATE ON tracks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_tracks_geom
    BEFORE INSERT OR UPDATE ON tracks
    FOR EACH ROW EXECUTE FUNCTION update_track_geom();