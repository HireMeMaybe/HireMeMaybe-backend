package auth

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

// GetAccessToken is a helper function to obtain an access token for a user by simulating a login API call.
// It takes the testing object, database connection, username, and password as parameters.
// It returns the access token as a string and any error encountered during the process.
func GetAccessToken(
	t *testing.T,
	db *database.DBinstanceStruct,
	username string,
	password string,
) (string, error) {
	t.Helper()
	handler := NewLocalAuthHandler(db)
	rec, resp, err := utilities.SimulateAPICall(handler.LocalLoginHandler, "/login", http.MethodPost, map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		return "", err
	}
	if rec.Code != http.StatusOK {
		return "", fmt.Errorf("login Failed: status %d, body: %s", rec.Code, rec.Body.String())
	}
	if resp["access_token"] == nil {
		return "", fmt.Errorf("login Failed: no access_token in response: %s", rec.Body.String())
	}
	return resp["access_token"].(string), nil
}

type mockInternalGoogleUserInfo struct {
	model.GoogleUserInfo
	AuthCode       string `json:"-"`
	AccessToken    string `json:"-"`
	TokenExchanged bool   `json:"-"`
}

// MockOAuth2Server creates a mock OAuth2 server for testing
type MockOAuth2Server struct {
	Server           *httptest.Server
	Config           *oauth2.Config
	MockUserInfo     []*mockInternalGoogleUserInfo // Store pointers
	MockInfoEndpoint string
}

// NewMockOAuth2Server creates and starts a mock OAuth2 server
func NewMockOAuth2Server(userInfo []model.GoogleUserInfo) *MockOAuth2Server {
	mock := &MockOAuth2Server{
		MockUserInfo: []*mockInternalGoogleUserInfo{},
	}

	// Initialize users with unique auth codes and access tokens
	for i, user := range userInfo {
		mock.MockUserInfo = append(mock.MockUserInfo, &mockInternalGoogleUserInfo{
			GoogleUserInfo: user,
			AuthCode:       fmt.Sprintf("mock_auth_code_%s_%d", user.GID, i),
			AccessToken:    fmt.Sprintf("mock_access_token_%s_%d", user.GID, i),
			TokenExchanged: false,
		})
	}

	mux := http.NewServeMux()

	// Token exchange endpoint - exchanges code for access token
	mux.HandleFunc("/token", mock.handleToken)

	// User info endpoint - returns user information
	mux.HandleFunc("/userinfo", mock.handleUserInfo)

	mock.Server = httptest.NewServer(mux)

	// Configure OAuth2 config to use mock server
	mock.Config = &oauth2.Config{
		ClientID:     "mock_client_id",
		ClientSecret: "mock_client_secret",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.openid",
		},
		Endpoint: oauth2.Endpoint{
			TokenURL: mock.Server.URL + "/token",
		},
		RedirectURL: "http://localhost:8080/callback",
	}

	mock.MockInfoEndpoint = mock.Server.URL + "/userinfo"

	return mock
}

// handleToken handles token exchange requests
func (m *MockOAuth2Server) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	grantType := r.FormValue("grant_type")
	code := r.FormValue("code")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")

	// Validate parameters
	if grantType != "authorization_code" {
		http.Error(w, "Invalid grant_type", http.StatusBadRequest)
		return
	}

	// Find user by auth code
	var foundUser *mockInternalGoogleUserInfo
	for _, user := range m.MockUserInfo {
		if user.AuthCode == code {
			foundUser = user
			break
		}
	}

	if foundUser == nil {
		http.Error(w, "Invalid authorization code", http.StatusUnauthorized)
		return
	}

	if clientID != m.Config.ClientID || clientSecret != m.Config.ClientSecret {
		http.Error(w, "Invalid client credentials", http.StatusUnauthorized)
		return
	}

	foundUser.TokenExchanged = true

	// Return access token
	tokenResponse := map[string]interface{}{
		"access_token":  foundUser.AccessToken,
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": "mock_refresh_token",
		"scope":         strings.Join(m.Config.Scopes, " "),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenResponse)
}

// handleUserInfo handles user info requests
func (m *MockOAuth2Server) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate bearer token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing authorization header", http.StatusUnauthorized)
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
		return
	}

	token := parts[1]

	// Find user by access token
	var foundUser *mockInternalGoogleUserInfo
	for _, user := range m.MockUserInfo {
		if user.AccessToken == token {
			foundUser = user
			break
		}
	}

	if foundUser == nil {
		http.Error(w, "Invalid access token", http.StatusUnauthorized)
		return
	}

	// Return user info (just the GoogleUserInfo part)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(foundUser.GoogleUserInfo)
}

// Close shuts down the mock server
func (m *MockOAuth2Server) Close() {
	if m.Server != nil {
		m.Server.Close()
	}
}

// GetAuthCode returns the mock authorization code for a specific user by GID
func (m *MockOAuth2Server) GetAuthCode(gid string) (string, error) {
	for _, user := range m.MockUserInfo {
		if user.GID == gid {
			user.TokenExchanged = false
			return user.AuthCode, nil
		}
	}
	return "", fmt.Errorf("user with GID %s not found", gid)
}

// AddUserInfo adds a new user info to the mock server with auto-generated codes
func (m *MockOAuth2Server) AddUserInfo(userInfo model.GoogleUserInfo) {
	index := len(m.MockUserInfo)
	m.MockUserInfo = append(m.MockUserInfo, &mockInternalGoogleUserInfo{
		GoogleUserInfo: userInfo,
		AuthCode:       fmt.Sprintf("mock_auth_code_%s_%d", userInfo.GID, index),
		AccessToken:    fmt.Sprintf("mock_access_token_%s_%d", userInfo.GID, index),
		TokenExchanged: false,
	})
}

// IsUserTokenExchanged returns whether a specific user's token was exchanged
func (m *MockOAuth2Server) IsUserTokenExchanged(gid string) bool {
	for _, user := range m.MockUserInfo {
		if user.GID == gid {
			return user.TokenExchanged
		}
	}
	return false
}
