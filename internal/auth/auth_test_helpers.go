package auth

import (
	"HireMeMaybe-backend/internal/utilities"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

// Helper to perform login
func SimulateLoginRequest(loginHandler func(*gin.Context), credentials interface{}) (*httptest.ResponseRecorder, map[string]interface{}, string, error) {
	rec, resp, err := utilities.SimulateAPICall(loginHandler, "/login", http.MethodPost, credentials)
	if err != nil {
		return nil, nil, "", err
	}
	if rec.Code != http.StatusOK {
		return rec, resp, "", fmt.Errorf("Login Failed: status %d, body: %s", rec.Code, rec.Body.String())
	}
	if resp["access_token"] == nil {
		return rec, resp, "", fmt.Errorf("Login Failed: no access_token in response: %s", rec.Body.String())
	}

	return rec, resp, resp["access_token"].(string), nil
}
