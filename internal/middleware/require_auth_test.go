package middleware

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/database"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
)

var (
	midTestDB   *database.DBinstanceStruct
	midTeardown func(context.Context, ...testcontainers.TerminateOption) error
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	var err error
	midTeardown, midTestDB, err = database.GetTestDB()
	if err != nil {
		os.Exit(1)
	}
	code := m.Run()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if midTeardown != nil {
		_ = midTeardown(ctx)
	}
	os.Exit(code)
}

func protectedEngine() *gin.Engine {
	r := gin.New()
	r.GET("/protected", RequireAuth(midTestDB), func(c *gin.Context) {
		u, exist := c.Get("user")
		if !exist {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "user": u})
	})
	return r
}

// Use real login flow (SimulateLoginRequest) to obtain token
func getLoginToken(t *testing.T) string {
	t.Helper()
	handler := auth.NewLocalAuthHandler(midTestDB)
	_, _, token, err := auth.SimulateLoginRequest(handler.LocalLoginHandler, map[string]string{
		"username": database.TestUserCPSK1.Username,
		"password": database.TestSeedPassword,
	})
	assert.NoError(t, err) 
	return token
}

func TestRequireAuth_Success(t *testing.T) {
	engine := protectedEngine()
	token := getLoginToken(t)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, true, body["ok"])
}

func TestRequireAuth_NoHeader(t *testing.T) {
	engine := protectedEngine()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Contains(t, body["error"], "Invalid authorization header")
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	engine := protectedEngine()
	token, _, err := auth.GenerateTokenWithDuration(database.TestUserCPSK1.ID, -1*time.Minute, auth.JwtIssuer)
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Equal(t, "Access token expired", body["error"])
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	engine := protectedEngine()
	// Create a valid token then corrupt it (signature mismatch)
	validToken, _, err := auth.GenerateTokenWithDuration(database.TestUserCPSK1.ID, time.Hour, auth.JwtIssuer)
	assert.NoError(t, err)
	invalid := validToken + "x"

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+invalid)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Contains(t, body["error"], "Failed to validate token")
}

func TestRequireAuth_UnknownUser(t *testing.T) {
	engine := protectedEngine()
	randomID := uuid.New()
	token, _, err := auth.GenerateTokenWithDuration(randomID, time.Hour, auth.JwtIssuer)
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	// Current middleware reports DB retrieval error (500)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Contains(t, body["error"], "User not exist")

}

func TestRequireAuth_InvalidIssuer(t *testing.T) {
	engine := protectedEngine()
	token, _, err := auth.GenerateTokenWithDuration(database.TestCPSK1.UserID, time.Hour, "invalid-issuer")
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["error"], "Invalid token issuer")

}