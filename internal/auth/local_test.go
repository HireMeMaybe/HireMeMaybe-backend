package auth

import (
	"context"
	"fmt"
	"net/http"

	"os"
	"testing"
	"time"

	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/utilities"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
)

var testDB *database.DBinstanceStruct
var testTeardown func(context.Context, ...testcontainers.TerminateOption) error

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	var err error
	testTeardown, testDB, err = database.GetTestDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start test db: %v\n", err)
		os.Exit(1)
	}

	m.Run()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := testTeardown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "teardown error: %v\n", err)
	}
}

// Helper: validate access token in response and return claims.
func assertValidAccessToken(t *testing.T, resp map[string]interface{}) *jwt.RegisteredClaims {
	t.Helper()
	tokenStr, ok := resp["access_token"].(string)
	assert.True(t, ok, "access_token not a string")
	token, err := ValidatedToken(tokenStr)
	assert.NoError(t, err)
	assert.True(t, token.Valid)
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	assert.True(t, ok, "claims type mismatch")
	assert.NotEmpty(t, claims.Subject, "token subject empty")
	return claims
}

func TestRegisterCPSK(t *testing.T) {
	handler := NewLocalAuthHandler(testDB)

	payload := map[string]string{
		"username": "test_cpsk_user",
		"password": "password123",
		"role":     "cpsk",
	}
	rec, resp, err := utilities.SimulateAPICall(handler.LocalRegisterHandler, "/register", http.MethodPost, payload)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code, "unexpected status, body: %s", rec.Body.String())

	assert.Contains(t, resp, "access_token")

	claims := assertValidAccessToken(t, resp)

	// Optional user id match if user object present
	if uVal, has := resp["user"]; has {
		if uMap, ok := uVal.(map[string]interface{}); ok {
			if idVal, ok := uMap["id"].(string); ok {
				assert.Equal(t, idVal, claims.Subject)
			}
		}
	}
}

func TestRegisterCompany(t *testing.T) {
	handler := NewLocalAuthHandler(testDB)

	orig := os.Getenv("BYPASS_VERIFICATION")
	_ = os.Setenv("BYPASS_VERIFICATION", "true")
	defer func() { _ = os.Setenv("BYPASS_VERIFICATION", orig) }()

	payload := map[string]string{
		"username": "test_company_user",
		"password": "companyPass123",
		"role":     "company",
	}
	rec, resp, err := utilities.SimulateAPICall(handler.LocalRegisterHandler, "/register", http.MethodPost, payload)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code, "unexpected status, body: %s", rec.Body.String())

	assert.Contains(t, resp, "access_token")

	claims := assertValidAccessToken(t, resp)

	userVal, ok := resp["user"]
	assert.True(t, ok, "user key missing in response")
	userObj, ok := userVal.(map[string]interface{})
	assert.True(t, ok, "user object has wrong type")

	if idVal, ok := userObj["id"].(string); ok {
		assert.Equal(t, idVal, claims.Subject, "JWT subject should match user id")
	}

	if vs, ok := userObj["verified_status"].(string); ok {
		assert.NotEmpty(t, vs, "verified_status should be set when BYPASS_VERIFICATION=true")
	}
}

// New: password shorter than 8 chars
func TestRegisterPasswordTooShort(t *testing.T) {
	handler := NewLocalAuthHandler(testDB)

	payload := map[string]string{
		"username": "short_pwd_user",
		"password": "1234567", // 7 chars
		"role":     "cpsk",
	}
	rec, resp, err := utilities.SimulateAPICall(handler.LocalRegisterHandler, "/register", http.MethodPost, payload)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	errMsg, _ := resp["error"].(string)
	assert.Contains(t, errMsg, "Password should longer or equal to 8 characters")
}

// New: duplicate username using seeded user
func TestRegisterDuplicateUsername(t *testing.T) {
	handler := NewLocalAuthHandler(testDB)

	payload := map[string]string{
		"username": database.TestUserCPSK1.Username, // seeded username
		"password": "password123",
		"role":     "cpsk",
	}
	rec, resp, err := utilities.SimulateAPICall(handler.LocalRegisterHandler, "/register", http.MethodPost, payload)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	errMsg, _ := resp["error"].(string)
	assert.Equal(t, "Username already exist", errMsg)
}

// New: invalid role
func TestRegisterInvalidRole(t *testing.T) {
	handler := NewLocalAuthHandler(testDB)

	payload := map[string]string{
		"username": "invalid_role_user",
		"password": "password123",
		"role":     "admin", // not allowed
	}
	rec, resp, err := utilities.SimulateAPICall(handler.LocalRegisterHandler, "/register", http.MethodPost, payload)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	errMsg, _ := resp["error"].(string)
	assert.Contains(t, errMsg, "Username, password, and Role (Only 'cpsk' or 'company) must be provided")
}

func TestLoginCPSKSuccess(t *testing.T) {
	handler := NewLocalAuthHandler(testDB)
	payload := map[string]string{
		"username": database.TestUserCPSK1.Username,
		"password": database.TestSeedPassword,
	}
	rec, resp, err := utilities.SimulateAPICall(handler.LocalLoginHandler, "/login", http.MethodPost, payload)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	assert.Contains(t, resp, "access_token")

	claims := assertValidAccessToken(t, resp)
	userVal, ok := resp["user"]
	assert.True(t, ok)
	if uMap, ok := userVal.(map[string]interface{}); ok {
		if idVal, ok := uMap["id"].(string); ok {
			assert.Equal(t, idVal, claims.Subject)
		}
	}
}

func TestLoginCompanySuccess(t *testing.T) {
	handler := NewLocalAuthHandler(testDB)
	payload := map[string]string{
		"username": database.TestUserCompany1.Username,
		"password": database.TestSeedPassword,
	}
	rec, resp, err := utilities.SimulateAPICall(handler.LocalLoginHandler, "/login", http.MethodPost, payload)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	assert.Contains(t, resp, "access_token")

	claims := assertValidAccessToken(t, resp)
	userVal, ok := resp["user"]
	assert.True(t, ok)
	if uMap, ok := userVal.(map[string]interface{}); ok {
		if idVal, ok := uMap["id"].(string); ok {
			assert.Equal(t, idVal, claims.Subject)
		}
	}
}

func TestLoginWrongPassword(t *testing.T) {
	handler := NewLocalAuthHandler(testDB)
	payload := map[string]string{
		"username": database.TestUserCPSK1.Username,
		"password": "WrongPass999!",
	}
	rec, resp, err := utilities.SimulateAPICall(handler.LocalLoginHandler, "/login", http.MethodPost, payload)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	errMsg, _ := resp["error"].(string)
	assert.Equal(t, "Username or password is incorrect", errMsg)
}

func TestLoginUserNotFound(t *testing.T) {
	handler := NewLocalAuthHandler(testDB)
	payload := map[string]string{
		"username": "non_existent_user_xyz",
		"password": "SomePassword1!",
	}
	rec, resp, err := utilities.SimulateAPICall(handler.LocalLoginHandler, "/login", http.MethodPost, payload)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	errMsg, _ := resp["error"].(string)
	assert.Equal(t, "Username or password is incorrect", errMsg)
}
