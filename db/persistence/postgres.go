// Package persistence provides PostgreSQL database access for VIGIL
package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds database configuration
type Config struct {
	Host            string
	Port            int
	Database        string
	User            string
	Password        string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	HealthCheckPeriod time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Host:              "localhost",
		Port:              5432,
		Database:          "vigil",
		User:              "vigil",
		Password:          "",
		MaxConns:          100,
		MinConns:          10,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   30 * time.Minute,
		HealthCheckPeriod: 1 * time.Minute,
	}
}

// Database represents a database connection
type Database struct {
	pool *pgxpool.Pool
}

// NewDatabase creates a new database connection
func NewDatabase(config *Config) (*Database, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d database=%s user=%s password=%s",
		config.Host, config.Port, config.Database, config.User, config.Password,
	)

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	poolConfig.MaxConns = config.MaxConns
	poolConfig.MinConns = config.MinConns
	poolConfig.MaxConnLifetime = config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = config.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = config.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	return &Database{pool: pool}, nil
}

// Close closes the database connection
func (db *Database) Close() {
	db.pool.Close()
}

// Ping checks database connectivity
func (db *Database) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// Stats returns connection pool statistics
func (db *Database) Stats() *pgxpool.Stat {
	return db.pool.Stat()
}

// BeginTx starts a transaction
func (db *Database) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	return db.pool.BeginTx(ctx, opts)
}

// Query executes a query
func (db *Database) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return db.pool.Query(ctx, sql, args...)
}

// QueryRow executes a query that returns a single row
func (db *Database) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.pool.QueryRow(ctx, sql, args...)
}

// Exec executes a query that doesn't return rows
func (db *Database) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.pool.Exec(ctx, sql, args...)
}

// Track represents a track record
type Track struct {
	ID            string     `json:"id"`
	TrackNumber   string     `json:"track_number"`
	TrackID       string     `json:"track_id"`
	SourceSystem  string     `json:"source_system"`
	Latitude      float64    `json:"latitude"`
	Longitude     float64    `json:"longitude"`
	Altitude      float64    `json:"altitude"`
	VelocityX     float64    `json:"velocity_x"`
	VelocityY     float64    `json:"velocity_y"`
	VelocityZ     float64    `json:"velocity_z"`
	Identity      string     `json:"identity"`
	Quality       string     `json:"quality"`
	Confidence    float64    `json:"confidence"`
	TrackType     string     `json:"track_type"`
	ForceID       string     `json:"force_id"`
	Environment   string     `json:"environment"`
	FirstDetect   time.Time  `json:"first_detect"`
	LastUpdate    time.Time  `json:"last_update"`
}

// Alert represents an alert record
type Alert struct {
	ID              string     `json:"id"`
	AlertID         string     `json:"alert_id"`
	AlertType       string     `json:"alert_type"`
	Priority        string     `json:"priority"`
	Status          string     `json:"status"`
	TrackID         string     `json:"track_id"`
	TrackNumber     string     `json:"track_number"`
	Message         string     `json:"message"`
	SourceSystem    string     `json:"source_system"`
	EscalationLevel string     `json:"escalation_level"`
	CreatedAt       time.Time  `json:"created_at"`
	ExpiresAt       *time.Time `json:"expires_at"`
	AcknowledgedAt  *time.Time `json:"acknowledged_at"`
	AcknowledgedBy  string     `json:"acknowledged_by"`
}

// Event represents an event record
type Event struct {
	ID          string     `json:"id"`
	EventType   string     `json:"event_type"`
	EventSource string     `json:"event_source"`
	EventTime   time.Time  `json:"event_time"`
	TrackID     string     `json:"track_id"`
	AlertID     string     `json:"alert_id"`
	Data        string     `json:"data"`
	Severity    string     `json:"severity"`
}

// TrackRepository provides track database operations
type TrackRepository struct {
	db *Database
}

// NewTrackRepository creates a new track repository
func NewTrackRepository(db *Database) *TrackRepository {
	return &TrackRepository{db: db}
}

// Create creates a new track
func (r *TrackRepository) Create(ctx context.Context, track *Track) error {
	sql := `
		INSERT INTO tracks (
			track_number, track_id, source_system,
			latitude, longitude, altitude,
			velocity_x, velocity_y, velocity_z,
			identity, quality, confidence,
			track_type, force_id, environment,
			first_detect, last_update
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, sql,
		track.TrackNumber, track.TrackID, track.SourceSystem,
		track.Latitude, track.Longitude, track.Altitude,
		track.VelocityX, track.VelocityY, track.VelocityZ,
		track.Identity, track.Quality, track.Confidence,
		track.TrackType, track.ForceID, track.Environment,
		track.FirstDetect, track.LastUpdate,
	).Scan(&track.ID)

	return err
}

// Get retrieves a track by ID
func (r *TrackRepository) Get(ctx context.Context, id string) (*Track, error) {
	sql := `
		SELECT id, track_number, track_id, source_system,
			latitude, longitude, altitude,
			velocity_x, velocity_y, velocity_z,
			identity, quality, confidence,
			track_type, force_id, environment,
			first_detect, last_update
		FROM tracks WHERE id = $1
	`

	track := &Track{}
	err := r.db.QueryRow(ctx, sql, id).Scan(
		&track.ID, &track.TrackNumber, &track.TrackID, &track.SourceSystem,
		&track.Latitude, &track.Longitude, &track.Altitude,
		&track.VelocityX, &track.VelocityY, &track.VelocityZ,
		&track.Identity, &track.Quality, &track.Confidence,
		&track.TrackType, &track.ForceID, &track.Environment,
		&track.FirstDetect, &track.LastUpdate,
	)

	if err != nil {
		return nil, err
	}

	return track, nil
}

// GetByTrackNumber retrieves a track by track number
func (r *TrackRepository) GetByTrackNumber(ctx context.Context, trackNumber string) (*Track, error) {
	sql := `
		SELECT id, track_number, track_id, source_system,
			latitude, longitude, altitude,
			velocity_x, velocity_y, velocity_z,
			identity, quality, confidence,
			track_type, force_id, environment,
			first_detect, last_update
		FROM tracks WHERE track_number = $1
	`

	track := &Track{}
	err := r.db.QueryRow(ctx, sql, trackNumber).Scan(
		&track.ID, &track.TrackNumber, &track.TrackID, &track.SourceSystem,
		&track.Latitude, &track.Longitude, &track.Altitude,
		&track.VelocityX, &track.VelocityY, &track.VelocityZ,
		&track.Identity, &track.Quality, &track.Confidence,
		&track.TrackType, &track.ForceID, &track.Environment,
		&track.FirstDetect, &track.LastUpdate,
	)

	if err != nil {
		return nil, err
	}

	return track, nil
}

// Update updates a track
func (r *TrackRepository) Update(ctx context.Context, track *Track) error {
	sql := `
		UPDATE tracks SET
			latitude = $2, longitude = $3, altitude = $4,
			velocity_x = $5, velocity_y = $6, velocity_z = $7,
			identity = $8, quality = $9, confidence = $10,
			track_type = $11, force_id = $12, environment = $13,
			last_update = $14
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, sql,
		track.ID,
		track.Latitude, track.Longitude, track.Altitude,
		track.VelocityX, track.VelocityY, track.VelocityZ,
		track.Identity, track.Quality, track.Confidence,
		track.TrackType, track.ForceID, track.Environment,
		track.LastUpdate,
	)

	return err
}

// Delete deletes a track
func (r *TrackRepository) Delete(ctx context.Context, id string) error {
	sql := `DELETE FROM tracks WHERE id = $1`
	_, err := r.db.Exec(ctx, sql, id)
	return err
}

// ListActive retrieves active tracks (updated within last 5 minutes)
func (r *TrackRepository) ListActive(ctx context.Context) ([]*Track, error) {
	sql := `
		SELECT id, track_number, track_id, source_system,
			latitude, longitude, altitude,
			velocity_x, velocity_y, velocity_z,
			identity, quality, confidence,
			track_type, force_id, environment,
			first_detect, last_update
		FROM tracks
		WHERE last_update > NOW() - INTERVAL '5 minutes'
		ORDER BY last_update DESC
	`

	rows, err := r.db.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []*Track
	for rows.Next() {
		track := &Track{}
		err := rows.Scan(
			&track.ID, &track.TrackNumber, &track.TrackID, &track.SourceSystem,
			&track.Latitude, &track.Longitude, &track.Altitude,
			&track.VelocityX, &track.VelocityY, &track.VelocityZ,
			&track.Identity, &track.Quality, &track.Confidence,
			&track.TrackType, &track.ForceID, &track.Environment,
			&track.FirstDetect, &track.LastUpdate,
		)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, track)
	}

	return tracks, nil
}

// AlertRepository provides alert database operations
type AlertRepository struct {
	db *Database
}

// NewAlertRepository creates a new alert repository
func NewAlertRepository(db *Database) *AlertRepository {
	return &AlertRepository{db: db}
}

// Create creates a new alert
func (r *AlertRepository) Create(ctx context.Context, alert *Alert) error {
	sql := `
		INSERT INTO alerts (
			alert_id, alert_type, priority, status,
			track_id, track_number, message, source_system,
			escalation_level, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, sql,
		alert.AlertID, alert.AlertType, alert.Priority, alert.Status,
		alert.TrackID, alert.TrackNumber, alert.Message, alert.SourceSystem,
		alert.EscalationLevel, alert.CreatedAt, alert.ExpiresAt,
	).Scan(&alert.ID)

	return err
}

// Get retrieves an alert by ID
func (r *AlertRepository) Get(ctx context.Context, id string) (*Alert, error) {
	sql := `
		SELECT id, alert_id, alert_type, priority, status,
			track_id, track_number, message, source_system,
			escalation_level, created_at, expires_at,
			acknowledged_at, acknowledged_by
		FROM alerts WHERE id = $1
	`

	alert := &Alert{}
	err := r.db.QueryRow(ctx, sql, id).Scan(
		&alert.ID, &alert.AlertID, &alert.AlertType, &alert.Priority, &alert.Status,
		&alert.TrackID, &alert.TrackNumber, &alert.Message, &alert.SourceSystem,
		&alert.EscalationLevel, &alert.CreatedAt, &alert.ExpiresAt,
		&alert.AcknowledgedAt, &alert.AcknowledgedBy,
	)

	if err != nil {
		return nil, err
	}

	return alert, nil
}

// Acknowledge acknowledges an alert
func (r *AlertRepository) Acknowledge(ctx context.Context, id, acknowledgedBy string) error {
	sql := `
		UPDATE alerts SET
			status = 'acknowledged',
			acknowledged_at = NOW(),
			acknowledged_by = $2,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, sql, id, acknowledgedBy)
	return err
}

// Complete marks an alert as complete
func (r *AlertRepository) Complete(ctx context.Context, id string) error {
	sql := `
		UPDATE alerts SET
			status = 'complete',
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, sql, id)
	return err
}

// ListActive retrieves active alerts
func (r *AlertRepository) ListActive(ctx context.Context) ([]*Alert, error) {
	sql := `
		SELECT id, alert_id, alert_type, priority, status,
			track_id, track_number, message, source_system,
			escalation_level, created_at, expires_at,
			acknowledged_at, acknowledged_by
		FROM alerts
		WHERE status IN ('pending', 'acknowledged', 'processing')
		ORDER BY priority DESC, created_at ASC
	`

	rows, err := r.db.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*Alert
	for rows.Next() {
		alert := &Alert{}
		err := rows.Scan(
			&alert.ID, &alert.AlertID, &alert.AlertType, &alert.Priority, &alert.Status,
			&alert.TrackID, &alert.TrackNumber, &alert.Message, &alert.SourceSystem,
			&alert.EscalationLevel, &alert.CreatedAt, &alert.ExpiresAt,
			&alert.AcknowledgedAt, &alert.AcknowledgedBy,
		)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}