-- Migration number: 0002
-- Created at: 2024-04-05 16:00:00
-- Description: Re-create tables with data types fully compatible with the original Go backend.

-- This migration is DESTRUCTIVE. It drops the existing tables to fix the data types.
-- This is necessary because the original schema was fundamentally incorrect.

DROP TABLE IF EXISTS `cookies`;
DROP TABLE IF EXISTS `users`;

-- Re-create users table (schema was mostly correct, but we do it for consistency)
CREATE TABLE `users` (
    `id` INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    `api_key` TEXT NOT NULL UNIQUE,
    `role` TEXT NOT NULL,
    `sharing_enabled` INTEGER NOT NULL DEFAULT 0, -- Storing boolean as integer (0/1)
    `created_at` TEXT NOT NULL, -- Storing DATETIME as TEXT (ISO8601 string)
    `updated_at` TEXT NOT NULL -- Storing DATETIME as TEXT (ISO8601 string)
);

-- Re-create cookies table with Go-compatible data types
CREATE TABLE `cookies` (
    `id` INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    `user_id` INTEGER NOT NULL,
    `domain` TEXT NOT NULL,
    `name` TEXT NOT NULL,
    `value` TEXT NOT NULL,
    `path` TEXT NOT NULL,
    `expires` TEXT, -- Storing DATETIME as TEXT (ISO8601 string), allows NULL
    `http_only` INTEGER NOT NULL DEFAULT 0, -- Storing boolean as integer (0/1)
    `secure` INTEGER NOT NULL DEFAULT 0, -- Storing boolean as integer (0/1)
    `same_site` TEXT NOT NULL,
    `is_sharable` INTEGER NOT NULL DEFAULT 0, -- Storing boolean as integer (0/1)
    `last_updated_from_extension_at` TEXT NOT NULL, -- Storing DATETIME as TEXT (ISO8601 string)
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Re-create indexes
CREATE UNIQUE INDEX `users_api_key_key` ON `users`(`api_key`);
CREATE UNIQUE INDEX `cookies_user_id_domain_name_path_key` ON `cookies`(`user_id`, `domain`, `name`, `path`);
