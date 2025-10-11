package store

import "cookie-syncer/api/internal/model"

// Store defines the interface for database operations.
type Store interface {
	// User methods
	CreateUser(apiKey, role string) (*model.User, error)
	GetUserByAPIKey(apiKey string) (*model.User, error)
	GetUserByID(userID int64) (*model.User, error)
	// Bulk operations for admins
	CreateUsers(count int, role string) ([]*model.User, error)
	DeleteUsersByIDs(userIDs []int64) (int64, error)
	DeleteUsersByAPIKeys(apiKeys []string) (int64, error)
	RefreshUsersAPIKeys(userIDs []int64) ([]*model.User, error)
	// Single operation for a user to refresh their own key
	UpdateUserAPIKey(userID int64, newAPIKey string) error
	UpdateUserSharing(userID int64, enabled bool) error

	// Cookie methods
	SyncCookies(userID int64, cookies []*model.Cookie) error
	GetSharableCookiesByDomain(domain string) ([]*model.Cookie, error)
	GetCookiesByUserID(userID int64) ([]*model.Cookie, error)
	GetCookiesByDomain(userID int64, domain string) ([]*model.Cookie, error)
	GetCookieByName(userID int64, domain, name string) (*model.Cookie, error)

	// Admin methods
	SearchCookies(domain, name string) ([]*model.Cookie, error)
}
