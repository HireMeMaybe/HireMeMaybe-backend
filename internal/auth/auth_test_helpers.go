package auth

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/utilities"
	"fmt"
	"net/http"
	"testing"
)

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
		return "", fmt.Errorf("Login Failed: status %d, body: %s", rec.Code, rec.Body.String())
	}
	if resp["access_token"] == nil {
		return "", fmt.Errorf("Login Failed: no access_token in response: %s", rec.Body.String())
	}
	return resp["access_token"].(string), nil
}
