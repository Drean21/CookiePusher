-- Migration to add fields for the hybrid cookie storage model.
-- This ALTER TABLE approach correctly appends new columns to the existing 'users' table.

ALTER TABLE users ADD COLUMN cookies_json TEXT;
ALTER TABLE users ADD COLUMN last_synced_at TEXT;
