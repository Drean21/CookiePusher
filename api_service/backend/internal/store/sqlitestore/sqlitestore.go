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

// New creates a new SQLiteStore, opens the database connection, and initializes the schema.
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

	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("could not initialize database schema: %w", err)
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

// initSchema creates the necessary tables if they don't exist.
func (s *SQLiteStore) initSchema() error {
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
		UNIQUE(user_id, domain, name)
	);`

	_, err := s.db.Exec(usersTable)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(cookiesTable)
	return err
}

func (s *SQLiteStore) GetUserByAPIKey(apiKey string) (*model.User, error) {
	query := `SELECT id, api_key, role, created_at, updated_at FROM users WHERE api_key = ?`
	row := s.db.QueryRow(query, apiKey)
	var user model.User
	err := row.Scan(&user.ID, &user.APIKey, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("could not scan user row: %w", err)
	}
	return &user, nil
}

func (s *SQLiteStore) GetUserByID(userID int64) (*model.User, error) {
	query := `SELECT id, api_key, role, created_at, updated_at FROM users WHERE id = ?`
	row := s.db.QueryRow(query, userID)
	var user model.User
	err := row.Scan(&user.ID, &user.APIKey, &user.Role, &user.CreatedAt, &user.UpdatedAt)
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
	query := `INSERT INTO users (api_key, role, created_at, updated_at) VALUES (?, ?, ?, ?)`
	now := time.Now()
	res, err := s.db.Exec(query, apiKey, role, now, now)
	if err != nil {
		return nil, fmt.Errorf("could not insert user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("could not get last insert ID: %w", err)
	}
	return &model.User{ID: id, APIKey: apiKey, Role: role, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *SQLiteStore) CreateUsers(count int, role string) ([]*model.User, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("could not start transaction: %w", err)
	}
	defer tx.Rollback()

	query := `INSERT INTO users (api_key, role, created_at, updated_at) VALUES (?, ?, ?, ?)`
	stmt, err := tx.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("could not prepare statement: %w", err)
	}
	defer stmt.Close()

	var createdUsers []*model.User
	for i := 0; i < count; i++ {
		apiKey := uuid.New().String()
		now := time.Now()
		res, err := stmt.Exec(apiKey, role, now, now)
		if err != nil {
			return nil, fmt.Errorf("could not insert user %d: %w", i+1, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("could not get last insert ID for user %d: %w", i+1, err)
		}
		createdUsers = append(createdUsers, &model.User{ID: id, APIKey: apiKey, Role: role, CreatedAt: now, UpdatedAt: now})
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

func (s *SQLiteStore) SyncCookies(userID int64, cookies []*model.Cookie) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("could not start transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
	INSERT INTO cookies (user_id, domain, name, value, path, expires, http_only, secure, same_site, last_updated_from_extension_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(user_id, domain, name) DO UPDATE SET
		value = excluded.value,
		path = excluded.path,
		expires = excluded.expires,
		http_only = excluded.http_only,
		secure = excluded.secure,
		same_site = excluded.same_site,
		last_updated_from_extension_at = excluded.last_updated_from_extension_at;
	`
	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("could not prepare statement: %w", err)
	}
	defer stmt.Close()

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
			now,
		)
		if err != nil {
			log.Error().
				Err(err).
				Int64("user_id", userID).
				Str("domain", cookie.Domain).
				Str("name", cookie.Name).
				Str("value", cookie.Value).
				Interface("expires", cookie.Expires).
				Msg("Failed to execute statement for a cookie")
			return fmt.Errorf("could not execute statement for cookie %s/%s: %w", cookie.Domain, cookie.Name, err)
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) GetCookiesByUserID(userID int64) ([]*model.Cookie, error) {
	query := `SELECT id, user_id, domain, name, value, path, expires, http_only, secure, same_site, last_updated_from_extension_at FROM cookies WHERE user_id = ?`
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
			&c.Expires, &c.HTTPOnly, &c.Secure, &c.SameSite, &c.LastUpdatedFromExtensionAt,
		)
		if err != nil {
			return nil, fmt.Errorf("could not scan cookie row: %w", err)
		}
		cookies = append(cookies, &c)
	}
	return cookies, nil
}

func (s *SQLiteStore) GetCookiesByDomain(userID int64, domain string) ([]*model.Cookie, error) {
	query := `SELECT id, user_id, domain, name, value, path, expires, http_only, secure, same_site, last_updated_from_extension_at FROM cookies WHERE user_id = ? AND (domain = ? OR domain LIKE ?)`
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
			&c.Expires, &c.HTTPOnly, &c.Secure, &c.SameSite, &c.LastUpdatedFromExtensionAt,
		)
		if err != nil {
			return nil, fmt.Errorf("could not scan cookie row: %w", err)
		}
		cookies = append(cookies, &c)
	}
	return cookies, nil
}

func (s *SQLiteStore) GetCookieByName(userID int64, domain, name string) (*model.Cookie, error) {
	query := `SELECT id, user_id, domain, name, value, path, expires, http_only, secure, same_site, last_updated_from_extension_at FROM cookies WHERE user_id = ? AND domain = ? AND name = ?`
	row := s.db.QueryRow(query, userID, domain, name)
	var c model.Cookie
	err := row.Scan(
		&c.ID, &c.UserID, &c.Domain, &c.Name, &c.Value, &c.Path,
		&c.Expires, &c.HTTPOnly, &c.Secure, &c.SameSite, &c.LastUpdatedFromExtensionAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("cookie not found")
		}
		return nil, fmt.Errorf("could not scan cookie row: %w", err)
	}
	return &c, nil
}

func (s *SQLiteStore) SearchCookies(domain, name string) ([]*model.Cookie, error) {
	return nil, fmt.Errorf("not implemented")
}
