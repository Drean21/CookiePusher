package sqlitestore

import (
	"cookie-syncer/api/internal/model"
	"cookie-syncer/api/internal/store"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// SQLiteStore is a SQLite-backed implementation of the store.Store interface.
type SQLiteStore struct {
	db *sql.DB
}

// New creates a new SQLiteStore, opens the database connection, and initializes/migrates the schema.
func New(dataSourceName string) (store.Store, error) {
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

	s := &SQLiteStore{db: db}

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
		adminAPIKey := "admin-secret-key-change-me"
		_, err := s.CreateUser(adminAPIKey, "admin")
		if err != nil {
			return nil, fmt.Errorf("could not create default admin user: %w", err)
		}
		fmt.Printf("Database was empty. Created default admin user with API Key: %s\n", adminAPIKey)
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


// --- User Methods ---

func (s *SQLiteStore) GetUserByAPIKey(apiKey string) (*model.User, error) {
	query := `SELECT id, api_key, role, sharing_enabled, created_at, updated_at FROM users WHERE api_key = ?`
	row := s.db.QueryRow(query, apiKey)

	var user model.User
	err := row.Scan(&user.ID, &user.APIKey, &user.Role, &user.SharingEnabled, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("could not scan user row: %w", err)
	}
	return &user, nil
}
func (s *SQLiteStore) GetUserByID(userID int64) (*model.User, error) {
	query := `SELECT id, api_key, role, sharing_enabled, created_at, updated_at FROM users WHERE id = ?`
	row := s.db.QueryRow(query, userID)
	var user model.User
	err := row.Scan(&user.ID, &user.APIKey, &user.Role, &user.SharingEnabled, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("could not scan user row: %w", err)
	}
	return &user, nil
}

// CreateUser is a helper for creating a single user, used for initialization.
func (s *SQLiteStore) CreateUser(apiKey, role string) (*model.User, error) {
	query := `INSERT INTO users (api_key, role, sharing_enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`
	now := time.Now()
	// New users have sharing disabled by default.
	res, err := s.db.Exec(query, apiKey, role, false, now, now)
	if err != nil {
		return nil, fmt.Errorf("could not insert user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("could not get last insert ID: %w", err)
	}
	return &model.User{ID: id, APIKey: apiKey, Role: role, SharingEnabled: false, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *SQLiteStore) CreateUsers(count int, role string) ([]*model.User, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("could not start transaction: %w", err)
	}
	defer tx.Rollback()

	query := `INSERT INTO users (api_key, role, sharing_enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`
	stmt, err := tx.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("could not prepare statement: %w", err)
	}
	defer stmt.Close()

	var createdUsers []*model.User
	for i := 0; i < count; i++ {
		apiKey := uuid.New().String()
		now := time.Now()
		res, err := stmt.Exec(apiKey, role, false, now, now)
		if err != nil {
			return nil, fmt.Errorf("could not insert user %d: %w", i+1, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("could not get last insert ID for user %d: %w", i+1, err)
		}
		createdUsers = append(createdUsers, &model.User{ID: id, APIKey: apiKey, Role: role, SharingEnabled: false, CreatedAt: now, UpdatedAt: now})
	}

	return createdUsers, tx.Commit()
}

func (s *SQLiteStore) DeleteUsersByIDs(userIDs []int64) (int64, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}
	query := `DELETE FROM users WHERE id IN (?` + strings.Repeat(",?", len(userIDs)-1) + `)`
	args := make([]interface{}, len(userIDs))
	for i, id := range userIDs {
		args[i] = id
	}

	res, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("could not delete users by ids: %w", err)
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) DeleteUsersByAPIKeys(apiKeys []string) (int64, error) {
	if len(apiKeys) == 0 {
		return 0, nil
	}
	query := `DELETE FROM users WHERE api_key IN (?` + strings.Repeat(",?", len(apiKeys)-1) + `)`
	args := make([]interface{}, len(apiKeys))
	for i, key := range apiKeys {
		args[i] = key
	}
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("could not delete users by api keys: %w", err)
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) RefreshUsersAPIKeys(userIDs []int64) ([]*model.User, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("could not start transaction: %w", err)
	}
	defer tx.Rollback()
	
	var refreshedUsers []*model.User
	for _, id := range userIDs {
		newAPIKey := uuid.New().String()
		now := time.Now()
		query := `UPDATE users SET api_key = ?, updated_at = ? WHERE id = ?`
		res, err := tx.Exec(query, newAPIKey, now, id)
		if err != nil {
			return nil, fmt.Errorf("could not update api key for user %d: %w", id, err)
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("could not get rows affected for user %d: %w", id, err)
		}
		if rowsAffected == 0 {
			continue // User not found, just skip
		}
		refreshedUsers = append(refreshedUsers, &model.User{ID: id, APIKey: newAPIKey, UpdatedAt: now})
	}
	
	return refreshedUsers, tx.Commit()
}

func (s *SQLiteStore) UpdateUserAPIKey(userID int64, newAPIKey string) error {
	query := `UPDATE users SET api_key = ?, updated_at = ? WHERE id = ?`
	res, err := s.db.Exec(query, newAPIKey, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("could not update api key: %w", err)
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


// --- Cookie Methods ---

func (s *SQLiteStore) SyncCookies(userID int64, cookies []*model.Cookie) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("could not start transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Delete all existing cookies for the user to ensure a clean slate.
	deleteQuery := `DELETE FROM cookies WHERE user_id = ?;`
	if _, err := tx.Exec(deleteQuery, userID); err != nil {
		return fmt.Errorf("could not delete old cookies for user %d: %w", userID, err)
	}

	// 2. Prepare the statement for inserting new cookies.
	insertQuery := `
	INSERT INTO cookies (user_id, domain, name, value, path, expires, http_only, secure, same_site, is_sharable, last_updated_from_extension_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	stmt, err := tx.Prepare(insertQuery)
	if err != nil {
		return fmt.Errorf("could not prepare insert statement: %w", err)
	}
	defer stmt.Close()

	// 3. Insert all cookies from the sync payload.
	now := time.Now()
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
			now,
		)
		if err != nil {
			log.Error().
				Err(err).
				Int64("user_id", userID).
				Str("domain", cookie.Domain).
				Str("name", cookie.Name).
				Msg("Failed to execute insert statement for a cookie")
			return fmt.Errorf("could not execute insert for cookie %s/%s: %w", cookie.Domain, cookie.Name, err)
		}
	}

	// 4. If all operations are successful, commit the transaction.
	return tx.Commit()
}

func (s *SQLiteStore) GetCookiesByUserID(userID int64) ([]*model.Cookie, error) {
	query := `SELECT id, user_id, domain, name, value, path, expires, http_only, secure, same_site, is_sharable, last_updated_from_extension_at FROM cookies WHERE user_id = ?`
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("could not query cookies: %w", err)
	}
	defer rows.Close()

	var cookies []*model.Cookie
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
func (s *SQLiteStore) GetCookiesByDomain(userID int64, domain string) ([]*model.Cookie, error) {
	query := `SELECT id, user_id, domain, name, value, path, expires, http_only, secure, same_site, is_sharable, last_updated_from_extension_at FROM cookies WHERE user_id = ? AND (domain = ? OR domain LIKE ?)`
	likeDomain := "%." + domain
	rows, err := s.db.Query(query, userID, domain, likeDomain)
	if err != nil {
		return nil, fmt.Errorf("could not query cookies by domain: %w", err)
	}
	defer rows.Close()
	var cookies []*model.Cookie
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

func (s *SQLiteStore) GetCookieByName(userID int64, domain, name string) (*model.Cookie, error) {
	query := `SELECT id, user_id, domain, name, value, path, expires, http_only, secure, same_site, is_sharable, last_updated_from_extension_at FROM cookies WHERE user_id = ? AND domain = ? AND name = ?`
	row := s.db.QueryRow(query, userID, domain, name)
	var c model.Cookie
	err := row.Scan(
		&c.ID, &c.UserID, &c.Domain, &c.Name, &c.Value, &c.Path,
		&c.Expires, &c.HTTPOnly, &c.Secure, &c.SameSite, &c.IsSharable, &c.LastUpdatedFromExtensionAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("cookie not found")
		}
		return nil, fmt.Errorf("could not scan cookie row: %w", err)
	}
	return &c, nil
}

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

	var cookies []*model.Cookie
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
