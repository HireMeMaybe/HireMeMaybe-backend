// Package auth contains handler relate to log in and create user account
package auth

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	// Auto load .env file
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

// OauthLoginHandler struct holds the database connection and OAuth2 configuration for handling OAuth login.
type OauthLoginHandler struct {
	DB               *database.DBinstanceStruct
	OauthConfig      *oauth2.Config
	UserInfoEndpoint string
}

type code struct {
	Code string `json:"code" binding:"required"`
}

// NewOauthLoginHandler creates a new instance of OauthLoginHandler with the provided database connection and OAuth2 configuration.
func NewOauthLoginHandler(db *database.DBinstanceStruct, oauthConfig *oauth2.Config, userInfoEndpoint string) *OauthLoginHandler {
	return &OauthLoginHandler{
		DB:               db,
		OauthConfig:      oauthConfig,
		UserInfoEndpoint: userInfoEndpoint,
	}
}

func (h *OauthLoginHandler) getUserInfo(c *gin.Context) (model.GoogleUserInfo, error) {

	var code code
	var uInfo model.GoogleUserInfo

	// check does body has code
	if err := c.ShouldBindJSON(&code); err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: fmt.Sprintf("No authorization code provided: %v", err.Error()),
		})
		return uInfo, err
	}

	// Exchange code with google and get userinfo
	token, err := h.OauthConfig.Exchange(
		context.Background(),
		code.Code,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to receive token: %v", err.Error()),
		})
		return uInfo, err
	}

	client := h.OauthConfig.Client(context.Background(), token)
	resp, err := client.Get(h.UserInfoEndpoint)
	if err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to fetch user information: %v", err.Error()),
		})
		return uInfo, err
	}
	if resp.StatusCode != http.StatusOK {
		// Read response body for better error message
		var bodyBytes []byte
		if resp.Body != nil {
			bodyBytes, _ = io.ReadAll(resp.Body)
		}
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to fetch user information: status=%d body=%s", resp.StatusCode, string(bodyBytes)),
		})
		// return a clear error so caller doesn't continue with empty user info
		return uInfo, fmt.Errorf("userinfo endpoint returned status %d", resp.StatusCode)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Fatal("Failed to close response body")
		}
	}()

	err = json.NewDecoder(resp.Body).Decode(&uInfo)
	if err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to decode user info: %v", err.Error()),
		})
		return uInfo, err
	}
	// defensive logging: ensure GID is present
	if uInfo.GID == "" {
		log.Printf("warning: decoded Google user info has empty GID: %+v", uInfo)
	}
	return uInfo, nil
}

func (h *OauthLoginHandler) loginOrRegisterUser(userModel model.UserModel, uinfo model.GoogleUserInfo, c *gin.Context) {
	log.Printf("User Info: %+v\n", uinfo)

	var user model.User
	respStatus := http.StatusOK

	err := h.DB.Where("google_id = ?", uinfo.GID).First(&user).Error

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):

		userModel.FillGoogleInfo(uinfo)

		if err := h.DB.Create(userModel).Error; err != nil {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to create user: %v", err.Error()),
			})
			return
		}

		respStatus = http.StatusCreated
	case err == nil:

		if err := h.DB.Preload("User").Preload("User.Punishment").Where("user_id = ?", user.ID).First(userModel).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
					Error: "You already registered as a different user type",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to retrieve user data: %v", err.Error()),
			})
			return
		}
	default:
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Database error: %v", err.Error()),
		})
		return
	}

	var accessToken string

	// TODO: change this when implementing refresh token
	var _ string

	accessToken, _, err = GenerateStandardToken(userModel.GetID())
	if err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to generate access token: %s", err.Error()),
		})
		return
	}

	resp := userModel.GetLoginResponse(accessToken)

	c.JSON(respStatus, resp)
}
