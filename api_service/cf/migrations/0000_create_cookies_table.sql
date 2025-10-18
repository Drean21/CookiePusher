-- Migration number: 0000
-- Created at: 2024-04-05 12:00:00
-- Description: Create the initial cookies table

-- CreateTable
CREATE TABLE `cookies` (
    `id` INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    `domain` TEXT NOT NULL,
    `name` TEXT NOT NULL,
    `value` TEXT NOT NULL,
    `path` TEXT NOT NULL,
    `expires` INTEGER,
    `http_only` INTEGER NOT NULL DEFAULT 0,
    `secure` INTEGER NOT NULL DEFAULT 0,
    `same_site` TEXT NOT NULL,
    `is_sharable` INTEGER NOT NULL DEFAULT 0,
    `last_updated_from_extension_at` TEXT NOT NULL
);

-- CreateIndex: Add a unique constraint to prevent duplicate cookies
CREATE UNIQUE INDEX `cookies_domain_name_path_key` ON `cookies`(`domain`, `name`, `path`);
