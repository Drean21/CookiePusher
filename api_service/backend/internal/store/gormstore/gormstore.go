package gormstore

import (
	"cookie-syncer/api/internal/config"
	"cookie-syncer/api/internal/model"
	"cookie-syncer/api/internal/store"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"database/sql"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite" // Anonymous import for the pure Go SQLite driver
)

// GormStore implements the store.Store interface using GORM.
type GormStore struct {
	db       *gorm.DB
	adminKey string
	poolKey  string
}

// New creates a new GormStore instance and connects to the database.
func New(cfg *config.Config, adminKey, poolKey string) (store.Store, error) {
	var dialector gorm.Dialector

	var db *gorm.DB
	var err error

	switch cfg.DBType {
	case "postgres":
		dialector = postgres.Open(cfg.DSN)
	case "mysql":
		dialector = mysql.Open(cfg.DSN)
	case "sqlite":
		// Use the pure-go driver directly
		sqlDB, err := sql.Open("sqlite", cfg.DSN)
		if err != nil {
			return nil, fmt.Errorf("failed to open sqlite database: %w", err)
		}
		dialector = sqlite.Dialector{
			Conn: sqlDB,
		}
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.DBType)
	}

	// Configure GORM logger to use zerolog
	var gormLogLevel logger.LogLevel
	switch cfg.GormLogLevel {
	case "info":
		gormLogLevel = logger.Info
	case "warn":
		gormLogLevel = logger.Warn
	case "error":
		gormLogLevel = logger.Error
	default:
		gormLogLevel = logger.Silent
	}

	gormLogger := logger.New(
		&log.Logger,
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  gormLogLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	if db == nil { // If not already opened by sqlite special case
		db, err = gorm.Open(dialector, &gorm.Config{
			Logger: gormLogger,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
	}

	// Get underlying sqlDB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	// Set max open connections
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConnections)
	// Set max idle connections
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConnections)

	s := &GormStore{db: db, adminKey: adminKey, poolKey: poolKey}

	// Run auto-migration
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("could not migrate database schema: %w", err)
	}

	// Check and create default admin user if the database is empty
	var userCount int64
	if err := s.db.Model(&model.User{}).Count(&userCount).Error; err != nil {
		return nil, fmt.Errorf("could not query user count: %w", err)
	}

	if userCount == 0 {
		defaultAPIKey := uuid.New().String()
		defaultUser := model.User{
			APIKey:         defaultAPIKey,
			Remark:         stringPtr("Default user"),
			SharingEnabled: false,
		}
		if err := s.db.Create(&defaultUser).Error; err != nil {
			return nil, fmt.Errorf("could not create default user: %w", err)
		}
		log.Info().Str("api_key", defaultAPIKey).Int64("user_id", defaultUser.ID).Msg("Database was empty. Created default user")
	}

	return s, nil
}

// migrate runs GORM's AutoMigrate for all models.
func (s *GormStore) migrate() error {
	if s.db.Dialector.Name() == "sqlite" {
		return s.migrateSQLite()
	}
	// For other databases like postgres, rely on AutoMigrate
	return s.db.AutoMigrate(&model.User{}, &model.Cookie{})
}

func (s *GormStore) migrateSQLite() error {
	// 1. Create meta table if it doesn't exist
	if err := s.db.Exec(`CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT);`).Error; err != nil {
		return fmt.Errorf("could not create meta table: %w", err)
	}

	// 2. Get current schema version
	var version int
	var meta struct {
		Key   string
		Value string
	}
	s.db.Table("meta").Where("key = ?", "version").First(&meta)
	fmt.Sscan(meta.Value, &version)

	// 3. Apply migrations in order
	if version == 0 {
		// New database, create schema from scratch
		log.Info().Msg("New database detected, running initial schema creation...")
		if err := s.migrationInit(); err != nil {
			return err
		}
		if err := s.setVersion(5); err != nil { // Set to the latest version
			return err
		}
		log.Info().Msg("Initial schema creation successful.")
	} else {
		// Existing database, apply incremental migrations
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
			log.Info().Msg("Running migration v5: Add cookies_json and last_synced_at to users table...")
			if err := s.migrationV5(); err != nil {
				return err
			}
			if err := s.setVersion(5); err != nil {
				return err
			}
			log.Info().Msg("Migration v5 successful.")
		}
	}

	return nil
}

func (s *GormStore) setVersion(version int) error {
	return s.db.Exec(`INSERT OR REPLACE INTO meta (key, value) VALUES ('version', ?)`, version).Error
}

func (s *GormStore) migrationInit() error {
	usersTable := `
	CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		api_key TEXT NOT NULL UNIQUE,
		remark TEXT,
		sharing_enabled BOOLEAN NOT NULL DEFAULT 0,
		last_synced_at DATETIME,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		deleted_at DATETIME,
		cookies_json TEXT
	);`

	cookiesTable := `
	CREATE TABLE cookies (
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

	if err := s.db.Exec(usersTable).Error; err != nil {
		return err
	}
	if err := s.db.Exec(cookiesTable).Error; err != nil {
		return err
	}
	return nil
}

func (s *GormStore) migrationV1() error {
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

	if err := s.db.Exec(usersTable).Error; err != nil {
		return err
	}
	if err := s.db.Exec(cookiesTable).Error; err != nil {
		return err
	}
	return nil
}

func (s *GormStore) migrationV2() error {
	// Add sharing_enabled to users table
	if err := s.db.Exec(`ALTER TABLE users ADD COLUMN sharing_enabled BOOLEAN NOT NULL DEFAULT 0;`).Error; err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("could not add sharing_enabled to users: %w", err)
		}
	}
	// Add is_sharable to cookies table
	if err := s.db.Exec(`ALTER TABLE cookies ADD COLUMN is_sharable BOOLEAN NOT NULL DEFAULT 0;`).Error; err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return fmt.Errorf("could not add is_sharable to cookies: %w", err)
		}
	}
	return nil
}

func (s *GormStore) migrationV3() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
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
		if err := tx.Exec(cookiesTableNew).Error; err != nil {
			return fmt.Errorf("v3: could not create new cookies table: %w", err)
		}

		copyData := `
		INSERT OR IGNORE INTO cookies_new (id, user_id, domain, name, value, path, expires, http_only, secure, same_site, is_sharable, last_updated_from_extension_at)
		SELECT id, user_id, domain, name, value, path, expires, http_only, secure, same_site, is_sharable, last_updated_from_extension_at
		FROM cookies;
		`
		if err := tx.Exec(copyData).Error; err != nil {
			log.Warn().Err(err).Msg("v3: Some data might not have been copied due to new constraints, which is expected.")
		}

		if err := tx.Exec(`DROP TABLE cookies;`).Error; err != nil {
			return fmt.Errorf("v3: could not drop old cookies table: %w", err)
		}

		if err := tx.Exec(`ALTER TABLE cookies_new RENAME TO cookies;`).Error; err != nil {
			return fmt.Errorf("v3: could not rename new cookies table: %w", err)
		}
		return nil
	})
}

func (s *GormStore) migrationV4() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		usersTableNew := `
		CREATE TABLE users_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api_key TEXT NOT NULL UNIQUE,
			remark TEXT,
			sharing_enabled BOOLEAN NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);`
		if err := tx.Exec(usersTableNew).Error; err != nil {
			return fmt.Errorf("v4: could not create new users table: %w", err)
		}

		copyData := `
		INSERT INTO users_new (id, api_key, sharing_enabled, created_at, updated_at)
		SELECT id, api_key, sharing_enabled, created_at, updated_at
		FROM users;
		`
		if err := tx.Exec(copyData).Error; err != nil {
			log.Warn().Err(err).Msg("v4: Error copying user data. This might happen if the old table structure is unexpected.")
			copyDataSimple := `
			INSERT INTO users_new (id, api_key, created_at, updated_at)
			SELECT id, api_key, created_at, updated_at
			FROM users;
			`
			if err2 := tx.Exec(copyDataSimple).Error; err2 != nil {
				return fmt.Errorf("v4: could not copy data to new users table, even with simplified schema: %w", err2)
			}
		}

		if err := tx.Exec(`DROP TABLE users;`).Error; err != nil {
			return fmt.Errorf("v4: could not drop old users table: %w", err)
		}

		if err := tx.Exec(`ALTER TABLE users_new RENAME TO users;`).Error; err != nil {
			return fmt.Errorf("v4: could not rename new users table: %w", err)
		}
		return nil
	})
}

func (s *GormStore) migrationV5() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`ALTER TABLE users ADD COLUMN cookies_json TEXT;`).Error; err != nil {
			if !strings.Contains(err.Error(), "duplicate column name") {
				return fmt.Errorf("v5: could not add cookies_json column to users: %w", err)
			}
		}
		if err := tx.Exec(`ALTER TABLE users ADD COLUMN last_synced_at DATETIME;`).Error; err != nil {
			if !strings.Contains(err.Error(), "duplicate column name") {
				return fmt.Errorf("v5: could not add last_synced_at column to users: %w", err)
			}
		}
		// Add deleted_at for soft delete support
		if err := tx.Exec(`ALTER TABLE users ADD COLUMN deleted_at DATETIME;`).Error; err != nil {
			if !strings.Contains(err.Error(), "duplicate column name") {
				return fmt.Errorf("v5: could not add deleted_at column to users: %w", err)
			}
		}
		return nil
	})
}

// --- Helper Functions ---

func stringPtr(s string) *string {
	return &s
}

func (s *GormStore) generateSafeAPIKey() string {
	for {
		apiKey := uuid.New().String()
		if apiKey != s.adminKey && apiKey != s.poolKey {
			return apiKey
		}
	}
}

// --- Store Interface Implementation ---

// User methods
func (s *GormStore) GetUserByAPIKey(apiKey string) (*model.User, error) {
	var user model.User
	if err := s.db.Where("api_key = ?", apiKey).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("could not get user by api key: %w", err)
	}
	return &user, nil
}

func (s *GormStore) GetUserByID(userID int64) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("could not get user by id: %w", err)
	}
	return &user, nil
}

func (s *GormStore) UpdateUserSharing(userID int64, enabled bool) error {
	result := s.db.Model(&model.User{}).Where("id = ?", userID).Update("sharing_enabled", enabled)
	if result.Error != nil {
		return fmt.Errorf("could not update user sharing status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (s *GormStore) GetUserSettings(userID int64) (*model.User, error) {
	return s.GetUserByID(userID)
}

// Admin methods
func (s *GormStore) CreateUsers(remarks []string) ([]*model.User, error) {
	var createdUsers []*model.User
	for _, remarkStr := range remarks {
		newUser := model.User{
			APIKey:         s.generateSafeAPIKey(),
			Remark:         stringPtr(remarkStr),
			SharingEnabled: false,
		}
		if err := s.db.Create(&newUser).Error; err != nil {
			return nil, fmt.Errorf("could not create user with remark %s: %w", remarkStr, err)
		}
		createdUsers = append(createdUsers, &newUser)
	}
	return createdUsers, nil
}

func (s *GormStore) UpdateUserRemark(userID int64, remark *string) error {
	result := s.db.Model(&model.User{}).Where("id = ?", userID).Update("remark", remark)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (s *GormStore) UpdateUserRemarkByAPIKey(apiKey string, remark *string) error {
	result := s.db.Model(&model.User{}).Where("api_key = ?", apiKey).Update("remark", remark)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (s *GormStore) AdminUpdateUserAPIKey(userID int64) (*model.User, error) {
	newAPIKey := s.generateSafeAPIKey()
	if err := s.db.Model(&model.User{}).Where("id = ?", userID).Update("api_key", newAPIKey).Error; err != nil {
		return nil, err
	}
	return s.GetUserByID(userID)
}

func (s *GormStore) AdminUpdateUserAPIKeyByAPIKey(apiKey string) (*model.User, error) {
	user, err := s.GetUserByAPIKey(apiKey)
	if err != nil {
		return nil, err // User not found
	}
	newAPIKey := s.generateSafeAPIKey()
	if err := s.db.Model(&model.User{}).Where("id = ?", user.ID).Update("api_key", newAPIKey).Error; err != nil {
		return nil, err
	}
	return s.GetUserByID(user.ID)
}

// Cookie methods
func (s *GormStore) SyncCookies(userID int64, cookies []*model.Cookie) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// 1. Update the users table with the JSON blob and last_synced_at
		jsonData, err := json.Marshal(cookies)
		if err != nil {
			return fmt.Errorf("could not marshal cookies to JSON: %w", err)
		}
		if err := tx.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
			"cookies_json":   string(jsonData),
			"last_synced_at": &now,
			"updated_at":     &now,
		}).Error; err != nil {
			return fmt.Errorf("could not update users table with JSON blob: %w", err)
		}

		// 2. Delete all existing cookies for the user
		if err := tx.Where("user_id = ?", userID).Delete(&model.Cookie{}).Error; err != nil {
			return fmt.Errorf("could not delete old cookies for user %d: %w", userID, err)
		}

		// 3. Insert new cookies
		if len(cookies) > 0 {
			for _, c := range cookies {
				newCookie := &model.Cookie{
					// ID is omitted to let the database generate it
					UserID:                     userID,
					Domain:                     c.Domain,
					Name:                       c.Name,
					Value:                      c.Value,
					Path:                       c.Path,
					Expires:                    c.Expires,
					HTTPOnly:                   c.HTTPOnly,
					Secure:                     c.Secure,
					SameSite:                   c.SameSite,
					IsSharable:                 c.IsSharable,
					LastUpdatedFromExtensionAt: now,
				}
				if err := tx.Create(newCookie).Error; err != nil {
					return fmt.Errorf("could not insert cookie %s/%s: %w", newCookie.Domain, newCookie.Name, err)
				}
			}
		}

		return nil
	})
}

func (s *GormStore) GetCookiesByUserID(userID int64) ([]*model.Cookie, error) {
	var user model.User
	if err := s.db.Select("cookies_json").First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("could not get user for cookies: %w", err)
	}

	if user.CookiesJSON == nil || *user.CookiesJSON == "" {
		return []*model.Cookie{}, nil
	}

	var cookies []*model.Cookie
	if err := json.Unmarshal([]byte(*user.CookiesJSON), &cookies); err != nil {
		return nil, fmt.Errorf("could not unmarshal cookies from JSON: %w", err)
	}

	return cookies, nil
}

func (s *GormStore) GetCookiesByDomain(userID int64, domain string) ([]*model.Cookie, error) {
	allCookies, err := s.GetCookiesByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("could not get all cookies for user %d: %w", userID, err)
	}

	filteredCookies := make([]*model.Cookie, 0)
	for _, cookie := range allCookies {
		if cookie.Domain == domain || strings.HasSuffix(cookie.Domain, "."+domain) {
			filteredCookies = append(filteredCookies, cookie)
		}
	}

	return filteredCookies, nil
}

func (s *GormStore) GetSharableCookiesByDomain(domain string) ([]*model.Cookie, error) {
	var cookies []*model.Cookie
	likeDomain := "%." + domain
	if err := s.db.Table("cookies c").
		Select("c.*").
		Joins("INNER JOIN users u ON c.user_id = u.id").
		Where("u.sharing_enabled = ? AND c.is_sharable = ? AND (c.domain = ? OR c.domain LIKE ?)", true, true, domain, likeDomain).
		Find(&cookies).Error; err != nil {
		return nil, fmt.Errorf("could not query sharable cookies: %w", err)
	}
	return cookies, nil
}
