package memorystore

import (
	"cookie-syncer/api/internal/model"
	"cookie-syncer/api/internal/store"
	"fmt"
	"sync"
)

// MemoryStore is an in-memory implementation of the store.Store interface.
// It's intended for development and testing, not for production.
type MemoryStore struct {
	mu      sync.Mutex
	nextID  int64
	users   map[string]*model.User // Keyed by API Key
	cookies map[int64][]*model.Cookie // Keyed by User ID
}

// New creates a new MemoryStore and pre-seeds it with a default admin user.
func New() store.Store {
	s := &MemoryStore{
		nextID:  1,
		users:   make(map[string]*model.User),
		cookies: make(map[int64][]*model.Cookie),
	}
	// Pre-seed with a default admin user for testing
	adminAPIKey := "admin-secret-key"
	_, _ = s.CreateUser(adminAPIKey, "admin")
	fmt.Printf("Created default admin user with API Key: %s\n", adminAPIKey)

	return s
}

func (s *MemoryStore) CreateUser(apiKey, role string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[apiKey]; exists {
		return nil, fmt.Errorf("API key already exists")
	}

	user := &model.User{
		ID:     s.nextID,
		APIKey: apiKey,
		Role:   role,
	}
	s.users[apiKey] = user
	s.nextID++

	return user, nil
}

func (s *MemoryStore) GetUserByAPIKey(apiKey string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[apiKey]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (s *MemoryStore) SyncCookies(userID int64, cookies []*model.Cookie) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Implementation to be added...
	return fmt.Errorf("not implemented")
}

func (s *MemoryStore) GetCookiesByUserID(userID int64) ([]*model.Cookie, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Implementation to be added...
	return nil, fmt.Errorf("not implemented")
}

func (s *MemoryStore) GetCookiesByDomain(userID int64, domain string) ([]*model.Cookie, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Implementation to be added...
	return nil, fmt.Errorf("not implemented")
}

func (s *MemoryStore) GetCookieByName(userID int64, domain, name string) (*model.Cookie, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Implementation to be added...
	return nil, fmt.Errorf("not implemented")
}

func (s *MemoryStore) SearchCookies(domain, name string) ([]*model.Cookie, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Implementation to be added...
	return nil, fmt.Errorf("not implemented")
}
