package sqlitestore

import (
	"cookie-syncer/api/internal/model"
	"cookie-syncer/api/internal/store"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// SQLiteStore is a SQLite-backed implementation of the store.Store interface.
type SQLiteStore struct {
	db       *sql.DB
	adminKey string
	poolKey  string
}

// New creates a new SQLiteStore, opens the database connection, and initializes/migrates the schema.
func New(dataSourceName string, adminKey, poolKey string) (store.Store, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	// Enable WAL mode for better concurrency.
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("could not enable WAL mode: %w", err)
	}

	s := &SQLiteStore{db: db, adminKey: adminKey, poolKey: poolKey}

	// Run migrations to ensure schema is up to date.
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("could not migrate database schema: %w", err)
	}

	// Check and create default admin user if the database is empty
	var userCount int
	err = s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return nil, fmt.Errorf("could not query user count: %w", err)
	}

	if userCount == 0 {
		defaultAPIKey := uuid.New().String()
		// Since CreateUser now returns a single user, and we want to handle remarks,
		// we'll create a slice of remarks for the new CreateUsers function.
		remarks := []string{"Default user"}
		users, err := s.CreateUsers(remarks)
		if err != nil || len(users) == 0 {
			return nil, fmt.Errorf("could not create default user: %w", err)
		}
		defaultAPIKey = users[0].APIKey
		user := users[0]
		log.Info().Str("api_key", defaultAPIKey).Int64("user_id", user.ID).Msg("Database was empty. Created default user")
	}

	return s, nil
}

// migrate handles database schema migrations.
func (s *SQLiteStore) migrate() error {
	// 1. Create meta table if it doesn't exist
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT);`)
	if err != nil {
		return fmt.Errorf("could not create meta table: %w", err)
	}

	// 2. Get current schema version
	var version int
	err = s.db.QueryRow(`SELECT value FROM meta WHERE key = 'version'`).Scan(&version)
	if err != nil {
		if err == sql.ErrNoRows {
			version = 0 // Database is new or pre-versioning
		} else {
			return fmt.Errorf("could not get schema version: %w", err)
		}
	}
	
	// 3. Apply migrations in order
	if version < 1 {
		log.Info().Msg("Running migration v1: Initial schema creation...")
		if err := s.migrationV1(); err != nil {
			return err
		}
		if err := s.setVersion(1); err != nil {
			return err
		}
		log.Info().Msg("Migration v1 successful.")
	}

	if version < 2 {
		log.Info().Msg("Running migration v2: Add sharing features...")
		if err := s.migrationV2(); err != nil {
			return err
		}
		if err := s.setVersion(2); err != nil {
			return err
		}
		log.Info().Msg("Migration v2 successful.")
	}

	if version < 3 {
		log.Info().Msg("Running migration v3: Fix UNIQUE constraint on cookies table...")
		if err := s.migrationV3(); err != nil {
			return err
		}
		if err := s.setVersion(3); err != nil {
			return err
		}
		log.Info().Msg("Migration v3 successful.")
	}

	if version < 4 {
		log.Info().Msg("Running migration v4: Remove user roles and add remarks...")
		if err := s.migrationV4(); err != nil {
			return err
		}
		if err := s.setVersion(4); err != nil {
			return err
		}
		log.Info().Msg("Migration v4 successful.")
	}

	if version < 5 {
		log.Info().Msg("Running migration v5: Add cookies_json and last_synced_at to users table and migrate data...")
		if err := s.migrationV5(); err != nil {
			return err
		}
		if err := s.setVersion(5); err != nil {
			return err
		}
		log.Info().Msg("Migration v5 successful.")
	}

	return nil
}

func (s *SQLiteStore) setVersion(version int) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO meta (key, value) VALUES ('version', ?)`, version)
	return err
}

func (s *SQLiteStore) migrationV1() error {
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		api_key TEXT NOT NULL UNIQUE,
		role TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);`

	cookiesTable := `
	CREATE TABLE IF NOT EXISTS cookies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		domain TEXT NOT NULL,
		name TEXT NOT NULL,
		value TEXT NOT NULL,
		path TEXT,
		expires DATETIME,
		http_only BOOLEAN,
		secure BOOLEAN,
		same_site TEXT,
		last_updated_from_extension_at DATETIME NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
		UNIQUE(user_id, domain, name, path)
	);`

	if _, err := s.db.Exec(usersTable); err != nil { return err }
	if _, err := s.db.Exec(cookiesTable); err != nil { return err }
	return nil
}

func (s *SQLiteStore) migrationV2() error {
	// Add sharing_enabled to users table
	_, err := s.db.Exec(`ALTER TABLE users ADD COLUMN sharing_enabled BOOLEAN NOT NULL DEFAULT 0;`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("could not add sharing_enabled to users: %w", err)
	}

	// Add is_sharable to cookies table
	_, err = s.db.Exec(`ALTER TABLE cookies ADD COLUMN is_sharable BOOLEAN NOT NULL DEFAULT 0;`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("could not add is_sharable to cookies: %w", err)
	}

	return nil
}

func (s *SQLiteStore) migrationV3() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("could not start transaction for v3 migration: %w", err)
	}
	defer tx.Rollback()

	// 1. Create a new table with the correct schema
	cookiesTableNew := `
	CREATE TABLE IF NOT EXISTS cookies_new (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		domain TEXT NOT NULL,
		name TEXT NOT NULL,
		value TEXT NOT NULL,
		path TEXT,
		expires DATETIME,
		http_only BOOLEAN,
		secure BOOLEAN,
		same_site TEXT,
		is_sharable BOOLEAN NOT NULL DEFAULT 0,
		last_updated_from_extension_at DATETIME NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
		UNIQUE(user_id, domain, name, path)
	);`
	if _, err := tx.Exec(cookiesTableNew); err != nil {
		return fmt.Errorf("v3: could not create new cookies table: %w", err)
	}

	// 2. Copy data from the old table to the new table
	// We ignore errors here because some rows might violate the new unique constraint (though unlikely with path)
	// The important part is to preserve as much data as possible.
	copyData := `
	INSERT OR IGNORE INTO cookies_new (id, user_id, domain, name, value, path, expires, http_only, secure, same_site, is_sharable, last_updated_from_extension_at)
	SELECT id, user_id, domain, name, value, path, expires, http_only, secure, same_site, is_sharable, last_updated_from_extension_at
	FROM cookies;
	`
	if _, err := tx.Exec(copyData); err != nil {
		log.Warn().Err(err).Msg("v3: Some data might not have been copied due to new constraints, which is expected.")
	}
	
	// 3. Drop the old table
	if _, err := tx.Exec(`DROP TABLE cookies;`); err != nil {
		return fmt.Errorf("v3: could not drop old cookies table: %w", err)
	}

	// 4. Rename the new table to the original name
	if _, err := tx.Exec(`ALTER TABLE cookies_new RENAME TO cookies;`); err != nil {
		return fmt.Errorf("v3: could not rename new cookies table: %w", err)
	}

	return tx.Commit()
}

func (s *SQLiteStore) migrationV4() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("could not start transaction for v4 migration: %w", err)
	}
	defer tx.Rollback()

	// 1. Create a new users table without the 'role' column and with 'remark'
	usersTableNew := `
	CREATE TABLE users_new (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		api_key TEXT NOT NULL UNIQUE,
		remark TEXT,
		sharing_enabled BOOLEAN NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);`
	if _, err := tx.Exec(usersTableNew); err != nil {
		return fmt.Errorf("v4: could not create new users table: %w", err)
	}

	// 2. Copy data from the old table to the new table
	copyData := `
	INSERT INTO users_new (id, api_key, sharing_enabled, created_at, updated_at)
	SELECT id, api_key, sharing_enabled, created_at, updated_at
	FROM users;
	`
	if _, err := tx.Exec(copyData); err != nil {
		// If the old table doesn't have sharing_enabled, this will fail.
		// Let's try to be more robust by checking column existence, but that's too complex for a migration script.
		// The migrations run in order, so v2 should have added sharing_enabled.
		// So this should be fine.
		log.Warn().Err(err).Msg("v4: Error copying user data. This might happen if the old table structure is unexpected.")
		// We will attempt to copy without it if the above fails
		copyDataSimple := `
		INSERT INTO users_new (id, api_key, created_at, updated_at)
		SELECT id, api_key, created_at, updated_at
		FROM users;
		`
		if _, err2 := tx.Exec(copyDataSimple); err2 != nil {
			return fmt.Errorf("v4: could not copy data to new users table, even with simplified schema: %w", err2)
		}
	}

	// 3. Drop the old table
	if _, err := tx.Exec(`DROP TABLE users;`); err != nil {
		return fmt.Errorf("v4: could not drop old users table: %w", err)
	}

	// 4. Rename the new table to the original name
	if _, err := tx.Exec(`ALTER TABLE users_new RENAME TO users;`); err != nil {
		return fmt.Errorf("v4: could not rename new users table: %w", err)
	}

	return tx.Commit()
}

func (s *SQLiteStore) migrationV5() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("could not start transaction for v5 migration: %w", err)
	}
	defer tx.Rollback()

	// 1. Add new columns to users table
	if _, err := tx.Exec(`ALTER TABLE users ADD COLUMN cookies_json TEXT;`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("v5: could not add cookies_json column to users: %w", err)
		}
	}
	if _, err := tx.Exec(`ALTER TABLE users ADD COLUMN last_synced_at DATETIME;`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("v5: could not add last_synced_at column to users: %w", err)
		}
	}

	// 2. Data migration from cookies table to users.cookies_json
	// This part is complex. We need to fetch all cookies, group them by user,
	// serialize them to JSON, and update the corresponding user row.

	// First, get all user IDs.
	rows, err := tx.Query(`SELECT id FROM users`)
	if err != nil {
		return fmt.Errorf("v5: could not query user IDs for migration: %w", err)
	}
	defer rows.Close()

	var userIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("v5: could not scan user ID: %w", err)
		}
		userIDs = append(userIDs, id)
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("v5: error iterating user IDs: %w", err)
	}
	
	// Prepare update statement
	updateStmt, err := tx.Prepare(`UPDATE users SET cookies_json = ?, last_synced_at = ? WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("v5: could not prepare user update statement: %w", err)
	}
	defer updateStmt.Close()

	// For each user, get their cookies, serialize, and update.
	for _, userID := range userIDs {
		cookies, err := s.getCookiesByUserID(tx, userID) // Use a transaction-aware getter
		if err != nil {
			log.Warn().Err(err).Int64("user_id", userID).Msg("v5: Could not get cookies for user during migration, skipping.")
			continue
		}

		if len(cookies) == 0 {
			continue // No cookies to migrate for this user
		}

		jsonData, err := json.Marshal(cookies)
		if err != nil {
			log.Warn().Err(err).Int64("user_id", userID).Msg("v5: Could not marshal cookies to JSON for user, skipping.")
			continue
		}

		// Find the latest update time from cookies
		var latestUpdate time.Time
		for _, c := range cookies {
			if c.LastUpdatedFromExtensionAt.After(latestUpdate) {
				latestUpdate = c.LastUpdatedFromExtensionAt
			}
		}

		if _, err := updateStmt.Exec(string(jsonData), latestUpdate, userID); err != nil {
			log.Warn().Err(err).Int64("user_id", userID).Msg("v5: Could not update user row with JSON cookies, skipping.")
			continue
		}
	}


	return tx.Commit()
}


// --- User Methods ---

func (s *SQLiteStore) GetUserByAPIKey(apiKey string) (*model.User, error) {
	query := `SELECT id, api_key, remark, sharing_enabled, last_synced_at, created_at, updated_at FROM users WHERE api_key = ?`
	row := s.db.QueryRow(query, apiKey)

	var user model.User
	err := row.Scan(&user.ID, &user.APIKey, &user.Remark, &user.SharingEnabled, &user.LastSyncedAt, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("could not scan user row: %w", err)
	}
	return &user, nil
}
func (s *SQLiteStore) GetUserByID(userID int64) (*model.User, error) {
	query := `SELECT id, api_key, remark, sharing_enabled, last_synced_at, created_at, updated_at FROM users WHERE id = ?`
	row := s.db.QueryRow(query, userID)
	var user model.User
	err := row.Scan(&user.ID, &user.APIKey, &user.Remark, &user.SharingEnabled, &user.LastSyncedAt, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("could not scan user row: %w", err)
	}
	return &user, nil
}

// generateSafeAPIKey creates a new UUID and ensures it doesn't collide with system keys.
func (s *SQLiteStore) generateSafeAPIKey() string {
	for {
		apiKey := uuid.New().String()
		if apiKey != s.adminKey && apiKey != s.poolKey {
			return apiKey
		}
	}
}


// --- Admin Methods ---

func (s *SQLiteStore) CreateUsers(remarks []string) ([]*model.User, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("could not start transaction: %w", err)
	}
	defer tx.Rollback()

	query := `INSERT INTO users (api_key, remark, sharing_enabled, last_synced_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
	stmt, err := tx.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("could not prepare statement: %w", err)
	}
	defer stmt.Close()

	var createdUsers []*model.User
	for _, remarkStr := range remarks {
		apiKey := s.generateSafeAPIKey()
		now := time.Now()
		var remark sql.NullString
		if remarkStr != "" {
			remark = sql.NullString{String: remarkStr, Valid: true}
		}

		res, err := stmt.Exec(apiKey, remark, false, nil, now, now)
		if err != nil {
			return nil, fmt.Errorf("could not insert user with remark %s: %w", remarkStr, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("could not get last insert ID for user with remark %s: %w", remarkStr, err)
		}

		// To get the full user object back, we fetch it by ID.
		user, err := s.getUserByID(tx, id)
		if err != nil {
			return nil, fmt.Errorf("could not fetch created user with id %d: %w", id, err)
		}
		createdUsers = append(createdUsers, user)
	}

	return createdUsers, tx.Commit()
}

func (s *SQLiteStore) UpdateUserRemark(userID int64, remark *string) error {
	query := `UPDATE users SET remark = ?, updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, remark, time.Now(), userID)
	return err
}

func (s *SQLiteStore) UpdateUserRemarkByAPIKey(apiKey string, remark *string) error {
	query := `UPDATE users SET remark = ?, updated_at = ? WHERE api_key = ?`
	_, err := s.db.Exec(query, remark, time.Now(), apiKey)
	return err
}

func (s *SQLiteStore) AdminUpdateUserAPIKey(userID int64) (*model.User, error) {
	newAPIKey := s.generateSafeAPIKey()
	query := `UPDATE users SET api_key = ?, updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, newAPIKey, time.Now(), userID)
	if err != nil {
		return nil, err
	}
	return s.GetUserByID(userID)
}

func (s *SQLiteStore) AdminUpdateUserAPIKeyByAPIKey(apiKey string) (*model.User, error) {
	user, err := s.GetUserByAPIKey(apiKey)
	if err != nil {
		return nil, err // User not found
	}
	newAPIKey := s.generateSafeAPIKey()
	query := `UPDATE users SET api_key = ?, updated_at = ? WHERE id = ?`
	_, err = s.db.Exec(query, newAPIKey, time.Now(), user.ID)
	if err != nil {
		return nil, err
	}
	// Fetch the user again to get the updated timestamp
	return s.GetUserByID(user.ID)
}

func (s *SQLiteStore) UpdateUserSharing(userID int64, enabled bool) error {
	query := `UPDATE users SET sharing_enabled = ? WHERE id = ?`
	res, err := s.db.Exec(query, enabled, userID)
	if err != nil {
		return fmt.Errorf("could not update user sharing status: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (s *SQLiteStore) GetUserSettings(userID int64) (*model.User, error) {
	return s.GetUserByID(userID)
}


// --- Cookie Methods ---

func (s *SQLiteStore) SyncCookies(userID int64, cookies []*model.Cookie) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("could not start transaction for sync: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()

	// 1. Update the users table with the JSON blob and last_synced_at
	jsonData, err := json.Marshal(cookies)
	if err != nil {
		return fmt.Errorf("could not marshal cookies to JSON: %w", err)
	}

	updateUserQuery := `UPDATE users SET cookies_json = ?, last_synced_at = ? WHERE id = ?`
	if _, err := tx.Exec(updateUserQuery, string(jsonData), now, userID); err != nil {
		return fmt.Errorf("could not update users table with JSON blob: %w", err)
	}


	// 2. Delete all existing cookies for the user from the 'cookies' table.
	deleteQuery := `DELETE FROM cookies WHERE user_id = ?;`
	if _, err := tx.Exec(deleteQuery, userID); err != nil {
		return fmt.Errorf("could not delete old cookies for user %d: %w", userID, err)
	}

	// 3. Prepare the statement for re-inserting new cookies into the 'cookies' table.
	insertQuery := `
	INSERT INTO cookies (user_id, domain, name, value, path, expires, http_only, secure, same_site, is_sharable, last_updated_from_extension_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	stmt, err := tx.Prepare(insertQuery)
	if err != nil {
		return fmt.Errorf("could not prepare insert statement for cookies table: %w", err)
	}
	defer stmt.Close()

	// 4. Insert all cookies from the sync payload into the 'cookies' table.
	for _, cookie := range cookies {
		_, err := stmt.Exec(
			userID,
			cookie.Domain,
			cookie.Name,
			cookie.Value,
			cookie.Path,
			cookie.Expires,
			cookie.HTTPOnly,
			cookie.Secure,
			cookie.SameSite,
			cookie.IsSharable,
			now, // Use the same timestamp for all cookies in this sync operation
		)
		if err != nil {
			// Since we are in a transaction, the whole operation will be rolled back.
			log.Error().
				Err(err).
				Int64("user_id", userID).
				Str("domain", cookie.Domain).
				Str("name", cookie.Name).
				Msg("Failed to execute insert statement for a cookie in 'cookies' table")
			return fmt.Errorf("could not execute insert for cookie %s/%s: %w", cookie.Domain, cookie.Name, err)
		}
	}

	// 5. If all operations are successful, commit the transaction.
	return tx.Commit()
}

// getCookiesByUserID is a helper for migration that can accept a transaction.
func (s *SQLiteStore) getCookiesByUserID(q querier, userID int64) ([]*model.Cookie, error) {
	query := `SELECT id, user_id, domain, name, value, path, expires, http_only, secure, same_site, is_sharable, last_updated_from_extension_at FROM cookies WHERE user_id = ?`
	rows, err := q.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("could not query cookies: %w", err)
	}
	defer rows.Close()

	cookies := make([]*model.Cookie, 0)
	for rows.Next() {
		var c model.Cookie
		err := rows.Scan(
			&c.ID, &c.UserID, &c.Domain, &c.Name, &c.Value, &c.Path,
			&c.Expires, &c.HTTPOnly, &c.Secure, &c.SameSite, &c.IsSharable, &c.LastUpdatedFromExtensionAt,
		)
		if err != nil {
			return nil, fmt.Errorf("could not scan cookie row: %w", err)
		}
		cookies = append(cookies, &c)
	}
	return cookies, nil
}

type querier interface {
    Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// getUserByID is a helper for migration that can accept a transaction.
func (s *SQLiteStore) getUserByID(q querier, userID int64) (*model.User, error) {
	query := `SELECT id, api_key, remark, sharing_enabled, last_synced_at, created_at, updated_at FROM users WHERE id = ?`
	row := q.QueryRow(query, userID)
	var user model.User
	err := row.Scan(&user.ID, &user.APIKey, &user.Remark, &user.SharingEnabled, &user.LastSyncedAt, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("could not scan user row: %w", err)
	}
	return &user, nil
}

func (s *SQLiteStore) GetCookiesByUserID(userID int64) ([]*model.Cookie, error) {
	query := `SELECT cookies_json FROM users WHERE id = ?`
	row := s.db.QueryRow(query, userID)

	var jsonData sql.NullString
	if err := row.Scan(&jsonData); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("could not scan for cookies_json: %w", err)
	}

	if !jsonData.Valid || jsonData.String == "" {
		return []*model.Cookie{}, nil // No cookies, return empty slice
	}

	var cookies []*model.Cookie
	if err := json.Unmarshal([]byte(jsonData.String), &cookies); err != nil {
		return nil, fmt.Errorf("could not unmarshal cookies from JSON: %w", err)
	}

	return cookies, nil
}
func (s *SQLiteStore) GetCookiesByDomain(userID int64, domain string) ([]*model.Cookie, error) {
	// First, get all cookies for the user from the JSON blob.
	allCookies, err := s.GetCookiesByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("could not get all cookies for user %d: %w", userID, err)
	}

	// Now, filter them by domain in the application layer.
	// This is less efficient than a SQL query but is necessitated by the JSON storage model
	// for the primary source of truth. The 'cookies' table is now just a projection for sharing.
	filteredCookies := make([]*model.Cookie, 0)
	for _, cookie := range allCookies {
		if cookie.Domain == domain || strings.HasSuffix(cookie.Domain, "."+domain) {
			filteredCookies = append(filteredCookies, cookie)
		}
	}

	return filteredCookies, nil
}

// GetCookieByName has been removed as it's more efficient to fetch all cookies from JSON and filter in-app.

func (s *SQLiteStore) GetSharableCookiesByDomain(domain string) ([]*model.Cookie, error) {
	query := `
		SELECT c.id, c.user_id, c.domain, c.name, c.value, c.path, c.expires, c.http_only, c.secure, c.same_site, c.is_sharable, c.last_updated_from_extension_at
		FROM cookies c
		JOIN users u ON c.user_id = u.id
		WHERE u.sharing_enabled = 1
		  AND c.is_sharable = 1
		  AND (c.domain = ? OR c.domain LIKE ?)`
	likeDomain := "%." + domain
	rows, err := s.db.Query(query, domain, likeDomain)
	if err != nil {
		return nil, fmt.Errorf("could not query sharable cookies: %w", err)
	}
	defer rows.Close()

	cookies := make([]*model.Cookie, 0)
	for rows.Next() {
		var c model.Cookie
		err := rows.Scan(
			&c.ID, &c.UserID, &c.Domain, &c.Name, &c.Value, &c.Path,
			&c.Expires, &c.HTTPOnly, &c.Secure, &c.SameSite, &c.IsSharable, &c.LastUpdatedFromExtensionAt,
		)
		if err != nil {
			return nil, fmt.Errorf("could not scan sharable cookie row: %w", err)
		}
		cookies = append(cookies, &c)
	}
	return cookies, nil
}


func (s *SQLiteStore) SearchCookies(domain, name string) ([]*model.Cookie, error) {
	return nil, fmt.Errorf("not implemented")
}
