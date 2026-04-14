-- Migration: 002_alerts
-- Description: Rollback alerts and alert_recipients tables

-- Drop triggers
DROP TRIGGER IF EXISTS update_alert_recipients_updated_at ON alert_recipients;
DROP TRIGGER IF EXISTS update_alerts_updated_at ON alerts;

-- Drop indexes
DROP INDEX IF EXISTS idx_recipients_alert;
DROP INDEX IF EXISTS idx_recipients_status;
DROP INDEX IF EXISTS idx_alerts_alert_id;
DROP INDEX IF EXISTS idx_alerts_status;
DROP INDEX IF EXISTS idx_alerts_priority;
DROP INDEX IF EXISTS idx_alerts_created_at;
DROP INDEX IF EXISTS idx_alerts_track_id;
DROP INDEX IF EXISTS idx_alerts_source;

-- Drop tables
DROP TABLE IF EXISTS alert_recipients;
DROP TABLE IF EXISTS alerts;