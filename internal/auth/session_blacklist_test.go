package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewInMemoryBlacklistStore(t *testing.T) {
	store := NewInMemoryBlacklistStore()
	assert.NotNil(t, store)
	assert.NotNil(t, store.blacklist)
}

func TestAddToBlacklist(t *testing.T) {
	store := NewInMemoryBlacklistStore()
	jti := "test-token-id"
	exp := time.Now().Add(time.Hour)

	err := store.AddToBlacklist(jti, exp)
	assert.NoError(t, err)

	// Verify it was added
	store.mu.RLock()
	expTime, exists := store.blacklist[jti]
	store.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, exp, expTime)
}

func TestIsBlacklisted_NotInList(t *testing.T) {
	store := NewInMemoryBlacklistStore()
	jti := "non-existent-token"

	isBlacklisted, err := store.IsBlacklisted(jti)
	assert.NoError(t, err)
	assert.False(t, isBlacklisted)
}

func TestIsBlacklisted_InList(t *testing.T) {
	store := NewInMemoryBlacklistStore()
	jti := "blacklisted-token"
	exp := time.Now().Add(time.Hour)

	err := store.AddToBlacklist(jti, exp)
	assert.NoError(t, err)

	isBlacklisted, err := store.IsBlacklisted(jti)
	assert.NoError(t, err)
	assert.True(t, isBlacklisted)
}

func TestCleanUpExpired(t *testing.T) {
	store := NewInMemoryBlacklistStore()

	// Add expired tokens
	expiredJti1 := "expired-token-1"
	expiredJti2 := "expired-token-2"
	expiredTime := time.Now().Add(-time.Hour)

	err := store.AddToBlacklist(expiredJti1, expiredTime)
	assert.NoError(t, err)
	err = store.AddToBlacklist(expiredJti2, expiredTime)
	assert.NoError(t, err)

	// Add valid token
	validJti := "valid-token"
	validTime := time.Now().Add(time.Hour)
	err = store.AddToBlacklist(validJti, validTime)
	assert.NoError(t, err)

	// Verify all tokens are in the blacklist
	store.mu.RLock()
	assert.Len(t, store.blacklist, 3)
	store.mu.RUnlock()

	// Clean up expired tokens
	store.CleanUpExpired()

	// Verify only valid token remains
	store.mu.RLock()
	assert.Len(t, store.blacklist, 1)
	_, exists := store.blacklist[validJti]
	assert.True(t, exists)
	_, exists = store.blacklist[expiredJti1]
	assert.False(t, exists)
	_, exists = store.blacklist[expiredJti2]
	assert.False(t, exists)
	store.mu.RUnlock()
}

func TestAddToBlacklist_MultipleTokens(t *testing.T) {
	store := NewInMemoryBlacklistStore()
	exp := time.Now().Add(time.Hour)

	tokens := []string{"token1", "token2", "token3"}
	for _, jti := range tokens {
		err := store.AddToBlacklist(jti, exp)
		assert.NoError(t, err)
	}

	// Verify all tokens are blacklisted
	for _, jti := range tokens {
		isBlacklisted, err := store.IsBlacklisted(jti)
		assert.NoError(t, err)
		assert.True(t, isBlacklisted)
	}
}

func TestAddToBlacklist_UpdateExpiration(t *testing.T) {
	store := NewInMemoryBlacklistStore()
	jti := "test-token"

	// Add token with initial expiration
	exp1 := time.Now().Add(time.Hour)
	err := store.AddToBlacklist(jti, exp1)
	assert.NoError(t, err)

	// Update token with new expiration
	exp2 := time.Now().Add(2 * time.Hour)
	err = store.AddToBlacklist(jti, exp2)
	assert.NoError(t, err)

	// Verify expiration was updated
	store.mu.RLock()
	expTime, exists := store.blacklist[jti]
	store.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, exp2, expTime)
}

func TestCleanUpExpired_EmptyStore(t *testing.T) {
	store := NewInMemoryBlacklistStore()

	// Should not panic on empty store
	assert.NotPanics(t, func() {
		store.CleanUpExpired()
	})

	store.mu.RLock()
	assert.Len(t, store.blacklist, 0)
	store.mu.RUnlock()
}

func TestCleanUpExpired_AllExpired(t *testing.T) {
	store := NewInMemoryBlacklistStore()
	expiredTime := time.Now().Add(-time.Hour)

	// Add multiple expired tokens
	for i := 0; i < 5; i++ {
		jti := "expired-token-" + string(rune(i))
		err := store.AddToBlacklist(jti, expiredTime)
		assert.NoError(t, err)
	}

	// Clean up
	store.CleanUpExpired()

	// Verify all were removed
	store.mu.RLock()
	assert.Len(t, store.blacklist, 0)
	store.mu.RUnlock()
}

func TestCleanUpExpired_NoneExpired(t *testing.T) {
	store := NewInMemoryBlacklistStore()
	validTime := time.Now().Add(time.Hour)

	// Add multiple valid tokens
	for i := 0; i < 5; i++ {
		jti := "valid-token-" + string(rune(i))
		err := store.AddToBlacklist(jti, validTime)
		assert.NoError(t, err)
	}

	// Clean up
	store.CleanUpExpired()

	// Verify all remain
	store.mu.RLock()
	assert.Len(t, store.blacklist, 5)
	store.mu.RUnlock()
}

func TestConcurrentAccess(t *testing.T) {
	store := NewInMemoryBlacklistStore()
	exp := time.Now().Add(time.Hour)

	// Test concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			jti := "token-" + string(rune(id))
			err := store.AddToBlacklist(jti, exp)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			jti := "token-" + string(rune(id))
			_, err := store.IsBlacklisted(jti)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
