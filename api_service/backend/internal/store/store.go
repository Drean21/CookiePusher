package store

import "cookie-syncer/api/internal/model"

// Store defines the interface for database operations.
type Store interface {
	// User methods
	CreateUser(apiKey, role string) (*model.User, error)
	GetUserByAPIKey(apiKey string) (*model.User, error)

	// Cookie methods
	SyncCookies(userID int64, cookies []*model.Cookie) error
	GetCookiesByUserID(userID int64) ([]*model.Cookie, error)
	GetCookiesByDomain(userID int64, domain string) ([]*model.Cookie, error)
	GetCookieByName(userID int64, domain, name string) (*model.Cookie, error)

	// Admin methods
	SearchCookies(domain, name string) ([]*model.Cookie, error)
}
