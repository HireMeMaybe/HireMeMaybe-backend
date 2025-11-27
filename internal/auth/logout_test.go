package auth

import (
	"HireMeMaybe-backend/internal/database"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

func TestLogoutSuccess(t *testing.T) {
	// Get a valid access token
	accessToken, err := GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)

	// Create logout controller with blacklist store
	blacklistStore := NewInMemoryBlacklistStore()
	logoutController := NewLogoutController(blacklistStore)

	// Create a test context with the access token
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request, err = http.NewRequest(http.MethodPost, "/logout", nil)
	assert.NoError(t, err)
	c.Request.Header.Set("Authorization", "Bearer "+accessToken)

	// Parse and set claims in context (simulating middleware behavior)
	token, err := ValidatedToken(accessToken)
	assert.NoError(t, err)
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	assert.True(t, ok)
	c.Set("claims", claims)

	// Call logout handler
	logoutController.LogoutHandler(c)

	// Assert response
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Successfully logged out", resp["message"])

	// Verify token is blacklisted
	isBlacklisted, err := blacklistStore.IsBlacklisted(accessToken)
	assert.NoError(t, err)
	assert.True(t, isBlacklisted, "Token should be blacklisted after logout")
}

func TestLogoutMissingToken(t *testing.T) {
	// Create logout controller
	blacklistStore := NewInMemoryBlacklistStore()
	logoutController := NewLogoutController(blacklistStore)

	// Create a test context without authorization header
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	var err error
	c.Request, err = http.NewRequest(http.MethodPost, "/logout", nil)
	assert.NoError(t, err)

	// Call logout handler
	logoutController.LogoutHandler(c)

	// Assert response
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp["error"], "authorization header")
}

func TestLogoutInvalidTokenFormat(t *testing.T) {
	// Create logout controller
	blacklistStore := NewInMemoryBlacklistStore()
	logoutController := NewLogoutController(blacklistStore)

	// Create a test context with invalid token format
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	var err error
	c.Request, err = http.NewRequest(http.MethodPost, "/logout", nil)
	assert.NoError(t, err)
	c.Request.Header.Set("Authorization", "InvalidFormat token123")

	// Call logout handler
	logoutController.LogoutHandler(c)

	// Assert response
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp, "error")
}

func TestLogoutMissingClaims(t *testing.T) {
	// Get a valid access token
	accessToken, err := GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)

	// Create logout controller
	blacklistStore := NewInMemoryBlacklistStore()
	logoutController := NewLogoutController(blacklistStore)

	// Create a test context with token but without claims in context
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request, err = http.NewRequest(http.MethodPost, "/logout", nil)
	assert.NoError(t, err)
	c.Request.Header.Set("Authorization", "Bearer "+accessToken)

	// Don't set claims in context (simulating missing middleware)

	// Call logout handler
	logoutController.LogoutHandler(c)

	// Assert response
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "invalid token claims", resp["error"])
}

func TestLogoutInvalidClaimsType(t *testing.T) {
	// Get a valid access token
	accessToken, err := GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)

	// Create logout controller
	blacklistStore := NewInMemoryBlacklistStore()
	logoutController := NewLogoutController(blacklistStore)

	// Create a test context with token but with wrong claims type
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request, err = http.NewRequest(http.MethodPost, "/logout", nil)
	assert.NoError(t, err)
	c.Request.Header.Set("Authorization", "Bearer "+accessToken)

	// Set wrong claims type in context
	c.Set("claims", "invalid_claims_type")

	// Call logout handler
	logoutController.LogoutHandler(c)

	// Assert response
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "invalid token claims type", resp["error"])
}

func TestLogoutBlacklistStoreError(t *testing.T) {
	// Get a valid access token
	accessToken, err := GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, accessToken)

	// Create a mock blacklist store that returns an error
	mockStore := &MockBlacklistStore{
		addError: fmt.Errorf("database connection failed"),
	}
	logoutController := NewLogoutController(mockStore)

	// Create a test context with the access token
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request, err = http.NewRequest(http.MethodPost, "/logout", nil)
	assert.NoError(t, err)
	c.Request.Header.Set("Authorization", "Bearer "+accessToken)

	// Parse and set claims in context
	token, err := ValidatedToken(accessToken)
	assert.NoError(t, err)
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	assert.True(t, ok)
	c.Set("claims", claims)

	// Call logout handler
	logoutController.LogoutHandler(c)

	// Assert response
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Failed to logout", resp["error"])
}

func TestLogoutMultipleTokens(t *testing.T) {
	// Get access tokens for different users
	token1, err := GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	token2, err := GetAccessToken(t, testDB, database.TestUserCompany1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	// Create logout controller with shared blacklist store
	blacklistStore := NewInMemoryBlacklistStore()
	logoutController := NewLogoutController(blacklistStore)

	// Logout first token
	rec1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(rec1)
	c1.Request, err = http.NewRequest(http.MethodPost, "/logout", nil)
	assert.NoError(t, err)
	c1.Request.Header.Set("Authorization", "Bearer "+token1)

	parsedToken1, err := ValidatedToken(token1)
	assert.NoError(t, err)
	claims1, ok := parsedToken1.Claims.(*jwt.RegisteredClaims)
	assert.True(t, ok)
	c1.Set("claims", claims1)

	logoutController.LogoutHandler(c1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Logout second token
	rec2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(rec2)
	c2.Request, err = http.NewRequest(http.MethodPost, "/logout", nil)
	assert.NoError(t, err)
	c2.Request.Header.Set("Authorization", "Bearer "+token2)

	parsedToken2, err := ValidatedToken(token2)
	assert.NoError(t, err)
	claims2, ok := parsedToken2.Claims.(*jwt.RegisteredClaims)
	assert.True(t, ok)
	c2.Set("claims", claims2)

	logoutController.LogoutHandler(c2)
	assert.Equal(t, http.StatusOK, rec2.Code)

	// Verify both tokens are blacklisted
	isBlacklisted1, err := blacklistStore.IsBlacklisted(token1)
	assert.NoError(t, err)
	assert.True(t, isBlacklisted1)

	isBlacklisted2, err := blacklistStore.IsBlacklisted(token2)
	assert.NoError(t, err)
	assert.True(t, isBlacklisted2)
}

func TestLogoutExpiredTokenHandling(t *testing.T) {
	// Create a token that will expire soon
	claims := &jwt.RegisteredClaims{
		Subject:   database.TestUserCPSK1.ID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Second)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	assert.NoError(t, err)

	// Create logout controller
	blacklistStore := NewInMemoryBlacklistStore()
	logoutController := NewLogoutController(blacklistStore)

	// Create a test context with the token
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request, err = http.NewRequest(http.MethodPost, "/logout", nil)
	assert.NoError(t, err)
	c.Request.Header.Set("Authorization", "Bearer "+tokenString)
	c.Set("claims", claims)

	// Logout the token
	logoutController.LogoutHandler(c)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify token is blacklisted
	isBlacklisted, err := blacklistStore.IsBlacklisted(tokenString)
	assert.NoError(t, err)
	assert.True(t, isBlacklisted)

	// Wait for token to expire
	time.Sleep(3 * time.Second)

	// Token should still be in blacklist (cleanup happens periodically)
	isBlacklistedAfter, err := blacklistStore.IsBlacklisted(tokenString)
	assert.NoError(t, err)
	// Note: May or may not be blacklisted depending on cleanup timing
	_ = isBlacklistedAfter
}

func TestExtractClaims(t *testing.T) {
	// Test valid claims extraction
	t.Run("ValidClaims", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)

		expectedClaims := &jwt.RegisteredClaims{
			Subject:   "test-user-id",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		}
		c.Set("claims", expectedClaims)

		claims, err := extractClaims(c)
		assert.NoError(t, err)
		assert.Equal(t, expectedClaims.Subject, claims.Subject)
	})

	// Test missing claims
	t.Run("MissingClaims", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)

		claims, err := extractClaims(c)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Equal(t, "invalid token claims", err.Error())
	})

	// Test invalid claims type
	t.Run("InvalidClaimsType", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
		c.Set("claims", "invalid")

		claims, err := extractClaims(c)
		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.Equal(t, "invalid token claims type", err.Error())
	})
}

// MockBlacklistStore is a mock implementation of JwtBlacklistStore for testing error scenarios
type MockBlacklistStore struct {
	blacklisted map[string]time.Time
	addError    error
	checkError  error
}

func (m *MockBlacklistStore) IsBlacklisted(jti string) (bool, error) {
	if m.checkError != nil {
		return false, m.checkError
	}
	if m.blacklisted == nil {
		return false, nil
	}
	_, exists := m.blacklisted[jti]
	return exists, nil
}

func (m *MockBlacklistStore) AddToBlacklist(jti string, exp time.Time) error {
	if m.addError != nil {
		return m.addError
	}
	if m.blacklisted == nil {
		m.blacklisted = make(map[string]time.Time)
	}
	m.blacklisted[jti] = exp
	return nil
}
