package handler

import "sync"

// UserLockManager provides a mutex for each user to prevent race conditions
// during database writes for the same user.
type UserLockManager struct {
	mu    sync.Mutex
	locks map[int64]*sync.Mutex
}

// NewUserLockManager creates a new lock manager.
func NewUserLockManager() *UserLockManager {
	return &UserLockManager{
		locks: make(map[int64]*sync.Mutex),
	}
}

// Lock acquires a lock for a specific user ID.
func (m *UserLockManager) Lock(userID int64) {
	m.mu.Lock()
	// Check if a mutex for this user already exists.
	userMutex, ok := m.locks[userID]
	if !ok {
		// If not, create one.
		userMutex = &sync.Mutex{}
		m.locks[userID] = userMutex
	}
	m.mu.Unlock()

	// Now, acquire the specific lock for this user.
	userMutex.Lock()
}

// Unlock releases the lock for a specific user ID.
func (m *UserLockManager) Unlock(userID int64) {
	m.mu.Lock()
	// It's guaranteed that the lock exists if Unlock is called after Lock.
	if userMutex, ok := m.locks[userID]; ok {
		userMutex.Unlock()
	}
	m.mu.Unlock()
}
