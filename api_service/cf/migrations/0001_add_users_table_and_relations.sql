-- Migration number: 0001
-- Created at: 2024-04-05 14:00:00
-- Description: Create the users table and link it to the cookies table.

-- CreateTable: Users
CREATE TABLE `users` (
    `id` INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    `api_key` TEXT NOT NULL,
    `role` TEXT NOT NULL,
    `sharing_enabled` INTEGER NOT NULL DEFAULT 0,
    `created_at` TEXT NOT NULL,
    `updated_at` TEXT NOT NULL
);

-- CreateIndex: Unique API Key for Users
CREATE UNIQUE INDEX `users_api_key_key` ON `users`(`api_key`);

-- ModifyTable: Add user_id to cookies table
-- D1 does not support ADD COLUMN with FOREIGN KEY directly.
-- We must recreate the table. This is safe as the DB is currently empty.
DROP TABLE `cookies`;

CREATE TABLE `cookies` (
    `id` INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    `user_id` INTEGER NOT NULL,
    `domain` TEXT NOT NULL,
    `name` TEXT NOT NULL,
    `value` TEXT NOT NULL,
    `path` TEXT NOT NULL,
    `expires` INTEGER,
    `http_only` INTEGER NOT NULL DEFAULT 0,
    `secure` INTEGER NOT NULL DEFAULT 0,
    `same_site` TEXT NOT NULL,
    `is_sharable` INTEGER NOT NULL DEFAULT 0,
    `last_updated_from_extension_at` TEXT NOT NULL,
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE ON UPDATE CASCADE
);

-- CreateIndex: Adjust unique constraint to be per-user
CREATE UNIQUE INDEX `cookies_user_id_domain_name_path_key` ON `cookies`(`user_id`, `domain`, `name`, `path`);
