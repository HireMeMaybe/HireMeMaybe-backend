package auth

import (
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginOrRegisterUser_NewCPSKUser(t *testing.T) {
	// Create mock OAuth2 server with test user
	mockUser := model.GoogleUserInfo{
		GID:            "google_test_123",
		Email:          "test.cpsk@example.com",
		FirstName:      "Test",
		LastName:       "CPSK",
		ProfilePicture: "https://example.com/photo.jpg",
	}
	mockServer := NewMockOAuth2Server([]model.GoogleUserInfo{mockUser})
	defer mockServer.Close()

	// Create OAuth handler with mock server config
	handler := NewOauthLoginHandler(testDB, mockServer.Config, mockServer.MockInfoEndpoint)

	// Get auth code for the test user
	authCode, err := mockServer.GetAuthCode(mockUser.GID)
	assert.NoError(t, err)

	// Simulate OAuth login request
	body := map[string]string{
		"code": authCode,
	}

	rec, resp, err := utilities.SimulateAPICall(
		handler.CPSKGoogleLoginHandler,
		"/auth/google/cpsk",
		http.MethodPost,
		body,
	)

	// Assert response
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code, "Expected 201 Created for new user")
	assert.NotNil(t, resp["access_token"], "Access token should be present")
	assert.NotNil(t, resp["user"], "User data should be present")

	// Verify token was exchanged
	assert.True(t, mockServer.IsUserTokenExchanged(mockUser.GID))

	// Verify user was created in database
	var createdUser model.CPSKUser
	err = testDB.Preload("User").Where("user_id IN (SELECT id FROM users WHERE google_id = ?)", mockUser.GID).First(&createdUser).Error
	assert.NoError(t, err)
	assert.Equal(t, mockUser.GID, createdUser.User.GoogleID)
	assert.Equal(t, mockUser.Email, *createdUser.User.Email)
	assert.Equal(t, mockUser.FirstName, createdUser.FirstName)
	assert.Equal(t, mockUser.LastName, createdUser.LastName)
}

func TestLoginOrRegisterUser_ExistingCPSKUser(t *testing.T) {
	// Create existing user in database
	email := "existing@example.com"
	existingUser := model.User{
		GoogleID:       "google_existing_123",
		Email:          &email,
		ProfilePicture: "https://example.com/old.jpg",
		Role:           model.RoleCPSK,
	}
	testDB.Create(&existingUser)

	cpskUser := model.CPSKUser{
		UserID: existingUser.ID,
		User:   existingUser,
		EditableCPSKInfo: model.EditableCPSKInfo{
			FirstName: "Existing",
			LastName:  "User",
		},
	}
	testDB.Create(&cpskUser)

	// Create mock OAuth2 server with same user (updated info)
	mockUser := model.GoogleUserInfo{
		GID:            "google_existing_123",
		Email:          "existing@example.com",
		FirstName:      "Updated",
		LastName:       "Name",
		ProfilePicture: "https://example.com/new.jpg",
	}
	mockServer := NewMockOAuth2Server([]model.GoogleUserInfo{mockUser})
	defer mockServer.Close()

	// Create OAuth handler with mock server config
	handler := NewOauthLoginHandler(testDB, mockServer.Config, mockServer.MockInfoEndpoint)

	// Get auth code for the test user
	authCode, err := mockServer.GetAuthCode(mockUser.GID)
	assert.NoError(t, err)

	// Simulate OAuth login request
	body := map[string]string{
		"code": authCode,
	}

	rec, resp, err := utilities.SimulateAPICall(
		handler.CPSKGoogleLoginHandler,
		"/auth/google/cpsk",
		http.MethodPost,
		body,
	)

	// Assert response
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code, "Expected 200 OK for existing user")
	assert.NotNil(t, resp["access_token"], "Access token should be present")
	assert.NotNil(t, resp["user"], "User data should be present")

	// Verify token was exchanged
	assert.True(t, mockServer.IsUserTokenExchanged(mockUser.GID))

	// Verify user exists and wasn't duplicated
	var count int64
	testDB.Model(&model.User{}).Where("google_id = ?", mockUser.GID).Count(&count)
	assert.Equal(t, int64(1), count, "Should have exactly one user with this Google ID")
}

func TestLoginOrRegisterUser_NewCompanyUser(t *testing.T) {
	// Create mock OAuth2 server with test user
	mockUser := model.GoogleUserInfo{
		GID:            "google_company_123",
		Email:          "company@example.com",
		FirstName:      "Test",
		LastName:       "Company",
		ProfilePicture: "https://example.com/company.jpg",
	}
	mockServer := NewMockOAuth2Server([]model.GoogleUserInfo{mockUser})
	defer mockServer.Close()

	// Create OAuth handler with mock server config
	handler := NewOauthLoginHandler(testDB, mockServer.Config, mockServer.MockInfoEndpoint)

	// Get auth code for the test user
	authCode, err := mockServer.GetAuthCode(mockUser.GID)
	assert.NoError(t, err)

	// Simulate OAuth login request
	body := map[string]string{
		"code": authCode,
	}

	rec, resp, err := utilities.SimulateAPICall(
		handler.CompanyGoogleLoginHandler,
		"/auth/google/company",
		http.MethodPost,
		body,
	)

	// Assert response
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code, "Expected 201 Created for new company")
	assert.NotNil(t, resp["access_token"], "Access token should be present")
	assert.NotNil(t, resp["user"], "Company data should be present")

	// Verify token was exchanged
	assert.True(t, mockServer.IsUserTokenExchanged(mockUser.GID))

	// Verify company was created in database
	var createdCompany model.CompanyUser
	err = testDB.Preload("User").Where("user_id IN (SELECT id FROM users WHERE google_id = ?)", mockUser.GID).First(&createdCompany).Error
	assert.NoError(t, err)
	assert.Equal(t, mockUser.GID, createdCompany.User.GoogleID)
	assert.Equal(t, mockUser.Email, *createdCompany.User.Email)
	assert.Equal(t, model.RoleCompany, createdCompany.User.Role)
}

func TestLoginOrRegisterUser_MultipleUsers(t *testing.T) {
	// Count users before creating new ones
	var userCountBefore int64
	testDB.Model(&model.User{}).Count(&userCountBefore)

	// Create mock OAuth2 server with multiple users
	users := []model.GoogleUserInfo{
		{
			GID:       "google_user1",
			Email:     "user1@example.com",
			FirstName: "User",
			LastName:  "One",
		},
		{
			GID:       "google_user2",
			Email:     "user2@example.com",
			FirstName: "User",
			LastName:  "Two",
		},
		{
			GID:       "google_user3",
			Email:     "user3@example.com",
			FirstName: "User",
			LastName:  "Three",
		},
	}
	mockServer := NewMockOAuth2Server(users)
	defer mockServer.Close()

	// Create OAuth handler with mock server config
	handler := NewOauthLoginHandler(testDB, mockServer.Config, mockServer.MockInfoEndpoint)

	// Test login for each user
	for _, user := range users {
		authCode, err := mockServer.GetAuthCode(user.GID)
		assert.NoError(t, err)

		body := map[string]string{
			"code": authCode,
		}

		rec, resp, err := utilities.SimulateAPICall(
			handler.CPSKGoogleLoginHandler,
			"/auth/google/cpsk",
			http.MethodPost,
			body,
		)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.NotNil(t, resp["access_token"])
		assert.True(t, mockServer.IsUserTokenExchanged(user.GID))
	}

	// Count users after creating new ones
	var userCountAfter int64
	testDB.Model(&model.User{}).Count(&userCountAfter)

	// Verify exactly 3 users were added
	assert.Equal(t, userCountBefore+3, userCountAfter, "Should have added exactly 3 new users")

	// Verify the specific users were created
	var newUserCount int64
	testDB.Model(&model.User{}).Where("google_id IN ?", []string{"google_user1", "google_user2", "google_user3"}).Count(&newUserCount)
	assert.Equal(t, int64(3), newUserCount, "Should have 3 users with the expected Google IDs")
}

func TestLoginOrRegisterUser_InvalidAuthCode(t *testing.T) {
	// Create mock OAuth2 server
	mockUser := model.GoogleUserInfo{
		GID:   "google_test",
		Email: "test@example.com",
	}
	mockServer := NewMockOAuth2Server([]model.GoogleUserInfo{mockUser})
	defer mockServer.Close()

	// Create OAuth handler with mock server config
	handler := NewOauthLoginHandler(testDB, mockServer.Config, mockServer.MockInfoEndpoint)

	// Use invalid auth code
	body := map[string]string{
		"code": "invalid_auth_code_12345",
	}

	rec, _, err := utilities.SimulateAPICall(
		handler.CPSKGoogleLoginHandler,
		"/auth/google/cpsk",
		http.MethodPost,
		body,
	)

	// Should fail with bad request
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "Should return 400 for invalid auth code")

	// Verify token was NOT exchanged
	assert.False(t, mockServer.IsUserTokenExchanged(mockUser.GID))
}

func TestLoginOrRegisterUser_MissingAuthCode(t *testing.T) {
	// Create mock OAuth2 server
	mockServer := NewMockOAuth2Server([]model.GoogleUserInfo{})
	defer mockServer.Close()

	// Create OAuth handler with mock server config
	handler := NewOauthLoginHandler(testDB, mockServer.Config, mockServer.MockInfoEndpoint)

	// Send request with no code
	body := map[string]string{}

	rec, _, err := utilities.SimulateAPICall(
		handler.CPSKGoogleLoginHandler,
		"/auth/google/cpsk",
		http.MethodPost,
		body,
	)

	// Should fail with bad request
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "Should return 400 for missing auth code")
}
