-- This migration removes the 'role' column from the 'users' table,
-- simplifying the user model as admin-specific logic has been removed from the application.

-- Recreate the users table without the 'role' column
CREATE TABLE users_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    api_key TEXT NOT NULL UNIQUE,
    sharing_enabled INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Copy data from the old table to the new one
INSERT INTO users_new (id, api_key, sharing_enabled, created_at, updated_at)
SELECT id, api_key, sharing_enabled, created_at, updated_at FROM users;

-- Drop the old table
DROP TABLE users;

-- Rename the new table to the original name
ALTER TABLE users_new RENAME TO users;
