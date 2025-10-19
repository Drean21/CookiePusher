-- CookiePusher Final Schema (Squashed Migration)
-- This single file creates the final, definitive database schema for the application.

-- Drop tables if they exist to ensure a clean start.
DROP TABLE IF EXISTS `cookies`;
DROP TABLE IF EXISTS `users`;

-- Create the `users` table with all necessary fields, including the hybrid storage optimization.
CREATE TABLE `users` (
    `id` INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    `api_key` TEXT NOT NULL UNIQUE,
    `remark` TEXT,
    `sharing_enabled` INTEGER NOT NULL DEFAULT 0,
    `cookies_json` TEXT,
    `last_synced_at` TEXT,
    `created_at` TEXT NOT NULL,
    `updated_at` TEXT NOT NULL
);

-- Create the `cookies` table, which serves as a normalized representation for complex queries (e.g., sharing).
CREATE TABLE `cookies` (
    `id` INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    `user_id` INTEGER NOT NULL,
    `domain` TEXT NOT NULL,
    `name` TEXT NOT NULL,
    `value` TEXT NOT NULL,
    `path` TEXT NOT NULL,
    `expires` TEXT,
    `http_only` INTEGER NOT NULL DEFAULT 0,
    `secure` INTEGER NOT NULL DEFAULT 0,
    `same_site` TEXT NOT NULL,
    `is_sharable` INTEGER NOT NULL DEFAULT 0,
    `last_updated_from_extension_at` TEXT NOT NULL,
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Create necessary indexes for performance.
CREATE UNIQUE INDEX `users_api_key_key` ON `users`(`api_key`);
CREATE UNIQUE INDEX `cookies_user_id_domain_name_path_key` ON `cookies`(`user_id`, `domain`, `name`, `path`);
