package middleware

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
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

var testDB   *database.DBinstanceStruct

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	var err error
	var midTeardown func(context.Context, ...testcontainers.TerminateOption) error
	midTeardown, testDB, err = database.GetTestDB()
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
	r.GET("/protected", RequireAuth(testDB), checkUserHandler)
	return r
}

func checkUserHandler(c *gin.Context) {
	u, exist := c.Get("user")
	if !exist {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "user": u})
}

func getCheckRoleHandler(role ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, exist := c.Get("user")
		if !exist {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false})
			return
		}
		user := utilities.ExtractUser(c)
		if !utilities.Contains(role, user.Role) {
			c.JSON(http.StatusForbidden, gin.H{"ok": false, "error": "User doesn't have permission to access"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "user": u, "message": "Hello, " + user.Role})
	}
}

func TestRequireAuth_Success(t *testing.T) {
	engine := gin.New()
	engine.GET("/protected", RequireAuth(testDB), checkUserHandler)
	token, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

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

func TestCheckRole_NoRequireAuthBefore(t *testing.T) {
	engine := gin.New()
	engine.GET("/need-role", CheckRole(model.RoleCPSK), getCheckRoleHandler("cpsk"))
	req, _ := http.NewRequest(http.MethodGet, "/need-role", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["error"], "User information not provided")
}

func TestCheckRole_WrongRole(t *testing.T) {
	engine := gin.New()
	engine.GET("/need-role", RequireAuth(testDB), CheckRole(model.RoleCompany), getCheckRoleHandler("company"))
	token, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/need-role", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["error"], "User doesn't have permission to access")
}

func TestCheckRole_Success(t *testing.T) {
	engine := gin.New()
	engine.GET("/need-role", RequireAuth(testDB), CheckRole(model.RoleCPSK), getCheckRoleHandler("cpsk"))
	token, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/need-role", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["message"], "Hello, cpsk")
}

func TestCheckRole_MultipleRoleCheck(t *testing.T) {
	engine := gin.New()
	engine.GET("/need-role", RequireAuth(testDB), CheckRole(model.RoleCPSK, model.RoleAdmin), getCheckRoleHandler(model.RoleCPSK, model.RoleAdmin))

	// Test with CPSK user
	tokenCPSK, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	reqCPSK, _ := http.NewRequest(http.MethodGet, "/need-role", nil)
	reqCPSK.Header.Set("Authorization", "Bearer "+tokenCPSK)
	recCPSK := httptest.NewRecorder()
	engine.ServeHTTP(recCPSK, reqCPSK)

	assert.Equal(t, http.StatusOK, recCPSK.Code)
	var bodyCPSK map[string]interface{}
	assert.NoError(t, json.Unmarshal(recCPSK.Body.Bytes(), &bodyCPSK))
	assert.Contains(t, bodyCPSK["message"], "Hello, cpsk")

	// Test with Admin user
	tokenAdmin, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	reqAdmin, _ := http.NewRequest(http.MethodGet, "/need-role", nil)
	reqAdmin.Header.Set("Authorization", "Bearer "+tokenAdmin)
	recAdmin := httptest.NewRecorder()
	engine.ServeHTTP(recAdmin, reqAdmin)

	assert.Equal(t, http.StatusOK, recAdmin.Code)
	var bodyAdmin map[string]interface{}
	assert.NoError(t, json.Unmarshal(recAdmin.Body.Bytes(), &bodyAdmin))
	assert.Contains(t, bodyAdmin["message"], "Hello, admin")

	// Test with Company user (should be forbidden)
	tokenCompany, err := auth.GetAccessToken(t, testDB, database.TestUserCompany1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	reqCompany, _ := http.NewRequest(http.MethodGet, "/need-role", nil)
	reqCompany.Header.Set("Authorization", "Bearer "+tokenCompany)
	recCompany := httptest.NewRecorder()
	engine.ServeHTTP(recCompany, reqCompany)

	assert.Equal(t, http.StatusForbidden, recCompany.Code)
	var bodyCompany map[string]interface{}
	assert.NoError(t, json.Unmarshal(recCompany.Body.Bytes(), &bodyCompany))
	assert.Contains(t, bodyCompany["error"], "User doesn't have permission to access")
}