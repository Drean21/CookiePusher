package model

import (
	"time"
	"gorm.io/gorm"
)

// User represents a user in our system.
type User struct {
	ID          int64          `json:"id" gorm:"primaryKey"`
	APIKey      string         `json:"-" gorm:"uniqueIndex;not null"` // API Key is sensitive, "-" prevents it from being marshalled into JSON
	Remark      *string        `json:"remark,omitempty" gorm:"type:text"`
	SharingEnabled bool         `json:"sharing_enabled" gorm:"default:false;not null"`
	LastSyncedAt *time.Time    `json:"last_synced_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"` // For soft deletes
	CookiesJSON *string        `json:"-" gorm:"type:text"` // Stores cookies as a JSON blob
}

// Cookie represents a cookie synced by a user.
type Cookie struct {
	ID                         int64      `json:"id" gorm:"primaryKey"`
	UserID                     int64      `json:"user_id" gorm:"uniqueIndex:idx_cookie_key,priority:1;not null"`
	Domain                     string     `json:"domain" gorm:"uniqueIndex:idx_cookie_key,priority:2;not null"`
	Name                       string     `json:"name" gorm:"uniqueIndex:idx_cookie_key,priority:3;not null"`
	Value                      string     `json:"value" gorm:"type:text;not null"`
	Path                       string     `json:"path" gorm:"uniqueIndex:idx_cookie_key,priority:4;not null"`
	Expires                    *time.Time `json:"expires,omitempty"` // Use pointer to handle null/zero time
	HTTPOnly                   bool       `json:"http_only" gorm:"default:false;not null"`
	Secure                     bool       `json:"secure" gorm:"default:false;not null"`
	SameSite                   string     `json:"same_site" gorm:"type:varchar(16)"`
	IsSharable                 bool       `json:"is_sharable" gorm:"default:false;not null"`
	LastUpdatedFromExtensionAt time.Time  `json:"last_updated_from_extension_at" gorm:"not null"`
}

// TableName specifies the table name for the User model.
func (User) TableName() string {
	return "users"
}

// TableName specifies the table name for the Cookie model.
func (Cookie) TableName() string {
	return "cookies"
}
