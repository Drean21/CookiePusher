package store

import "cookie-syncer/api/internal/model"

// Store defines the interface for database operations.
type Store interface {
	// User methods
	GetUserByAPIKey(apiKey string) (*model.User, error)
	GetUserByID(userID int64) (*model.User, error)
	UpdateUserSharing(userID int64, enabled bool) error
	GetUserSettings(userID int64) (*model.User, error)

	// Admin methods
	CreateUsers(remarks []string) ([]*model.User, error)
	UpdateUserRemark(userID int64, remark *string) error
	UpdateUserRemarkByAPIKey(apiKey string, remark *string) error
	AdminUpdateUserAPIKey(userID int64) (*model.User, error)
	AdminUpdateUserAPIKeyByAPIKey(apiKey string) (*model.User, error)

	// Cookie methods
	SyncCookies(userID int64, cookies []*model.Cookie) error
	GetSharableCookiesByDomain(domain string) ([]*model.Cookie, error)
	GetCookiesByUserID(userID int64) ([]*model.Cookie, error)
	GetCookiesByDomain(userID int64, domain string) ([]*model.Cookie, error)
	// GetCookieByName(userID int64, domain, name string) (*model.Cookie, error) // Removed

	// SearchCookies(domain, name string) ([]*model.Cookie, error) // Not implemented, removed
}
