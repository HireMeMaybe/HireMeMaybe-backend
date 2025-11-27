package auth

import (
	"sync"
	"time"
)

// JwtBlacklistStore defines the interface for a JWT blacklist store.
type JwtBlacklistStore interface {
	// IsBlacklisted checks if the given JWT ID (jti) is blacklisted.
	IsBlacklisted(jti string) (bool, error)
	// AddToBlacklist adds the given JWT ID (jti) to the blacklist with an expiration time.
	AddToBlacklist(jti string, exp time.Time) error
}

// InMemoryBlacklistStore is an in-memory implementation of JwtBlacklistStore.
type InMemoryBlacklistStore struct {
	blacklist map[string]time.Time
	mu        sync.RWMutex
}

// NewInMemoryBlacklistStore creates a new instance of InMemoryBlacklistStore.
func NewInMemoryBlacklistStore() *InMemoryBlacklistStore {
	store := &InMemoryBlacklistStore{
		blacklist: make(map[string]time.Time),
	}
	go periodiclyCleanUp(store, time.Minute*5)
	return store
}

func periodiclyCleanUp(store *InMemoryBlacklistStore, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		store.CleanUpExpired()
	}
}

// CleanUpExpired removes expired JWT IDs from the blacklist.
func (s *InMemoryBlacklistStore) CleanUpExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for jti, exp := range s.blacklist {
		if exp.Before(now) {
			delete(s.blacklist, jti)
		}
	}
}

// IsBlacklisted checks if the given JWT ID (jti) is blacklisted.
func (s *InMemoryBlacklistStore) IsBlacklisted(jti string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.blacklist[jti]
	return exists, nil
}

// AddToBlacklist adds the given JWT ID (jti) to the blacklist with an expiration time.
func (s *InMemoryBlacklistStore) AddToBlacklist(jti string, exp time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.blacklist[jti] = exp
	return nil
}
