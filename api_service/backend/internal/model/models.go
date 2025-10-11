package model

import "time"

// User represents a user in our system.
type User struct {
	ID             int64     `json:"id"`
	APIKey         string    `json:"-"` // API Key is sensitive, "-" prevents it from being marshalled into JSON
	Role           string    `json:"role"`
	SharingEnabled bool      `json:"sharing_enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Cookie represents a cookie synced by a user.
type Cookie struct {
	ID                         int64      `json:"id"`
	UserID                     int64      `json:"user_id"`
	Domain                     string     `json:"domain"`
	Name                       string     `json:"name"`
	Value                      string     `json:"value"`
	Path                       string     `json:"path"`
	Expires                    *time.Time `json:"expires,omitempty"` // Use pointer to handle null/zero time
	HTTPOnly                   bool       `json:"http_only"`
	Secure                     bool       `json:"secure"`
	SameSite                   string     `json:"same_site"`
	IsSharable                 bool       `json:"is_sharable"`
	LastUpdatedFromExtensionAt time.Time  `json:"last_updated_from_extension_at"`
}
