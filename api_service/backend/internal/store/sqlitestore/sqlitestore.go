package sqlitestore

import (
	"cookie-syncer/api/internal/model"
	"cookie-syncer/api/internal/store"
	"database/sql"
	"fmt"
	"time"
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


// --- Interface method stubs ---

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

	return &model.User{
		ID:        id,
		APIKey:    apiKey,
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
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

func (s *SQLiteStore) SyncCookies(userID int64, cookies []*model.Cookie) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("could not start transaction: %w", err)
	}
	// Defer a rollback in case of error.
	// If the transaction is committed, this will be a no-op.
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
			return fmt.Errorf("could not execute statement for cookie %s/%s: %w", cookie.Domain, cookie.Name, err)
		}
	}

	// If all statements executed successfully, commit the transaction.
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
	return nil, fmt.Errorf("not implemented")
}

func (s *SQLiteStore) GetCookieByName(userID int64, domain, name string) (*model.Cookie, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *SQLiteStore) SearchCookies(domain, name string) ([]*model.Cookie, error) {
	return nil, fmt.Errorf("not implemented")
}
