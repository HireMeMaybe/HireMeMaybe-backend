// Package auth contains handler relate to log in and create user account
package auth

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	// Auto load .env file
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type OauthLoginHandler struct {
	DB *database.DBinstanceStruct
	OauthConfig *oauth2.Config
}

func NewOauthLoginHandler(db *database.DBinstanceStruct, oauthConfig *oauth2.Config) *OauthLoginHandler {
	return &OauthLoginHandler{
		DB: db,
		OauthConfig: oauthConfig,
	}
}

func (h *OauthLoginHandler) getUserInfo(c *gin.Context) (uInfo struct {
	GID            string `json:"sub"`
	FirstName      string `json:"given_name"`
	LastName       string `json:"family_name"`
	Email          string `json:"email"`
	ProfilePicture string `json:"picture"`
}, e error) {

	var code struct {
		Code string `json:"code" binding:"required"`
	}

	// check does body has code
	if err := c.ShouldBindJSON(&code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("No authorization code provided: %v", err.Error()),
		})
		return uInfo, err
	}

	// Exchange code with google and get userinfo
	token, err := h.OauthConfig.Exchange(
		context.Background(),
		code.Code,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to receive token: %v", err.Error()),
		})
		return uInfo, err
	}

	client := h.OauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to fetch user information: %v", err.Error()),
		})
		return uInfo, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Fatal("Failed to close response body")
		}
	}()

	err = json.NewDecoder(resp.Body).Decode(&uInfo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to decode user info: %v", err.Error()),
		})
		return uInfo, err
	}
	return uInfo, nil
}

// CPSKGoogleLoginHandler handles Google login authentication for cpsk role, exchanges code for user
// info, checks and creates user in the database, generates an access token, and returns user
// information with the access token.
func (h *OauthLoginHandler) CPSKGoogleLoginHandler(c *gin.Context) {

	uInfo, err := h.getUserInfo(c)
	if err != nil {
		return
	}

	respStatus := http.StatusOK

	// Check does user are already in DB or not
	var user model.User
	var cpskUser model.CPSKUser

	err = h.DB.Where("google_id = ?", uInfo.GID).First(&user).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		cpskUser = model.CPSKUser{
			User: model.User{
				Email:          &uInfo.Email,
				GoogleID:       uInfo.GID,
				Username:       uInfo.FirstName,
				Role:           model.RoleCPSK,
				ProfilePicture: uInfo.ProfilePicture,
			},
			FirstName: uInfo.FirstName,
			LastName:  uInfo.LastName,
		}

		if err := h.DB.Create(&cpskUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to create user: %v", err.Error()),
			})
			return
		}

		respStatus = http.StatusCreated
	case err == nil:
		if err := h.DB.Preload("User").Where("user_id = ?", user.ID).First(&cpskUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to retrieve user data: %v", err.Error()),
			})
			return
		}
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Database error: %v", err.Error()),
		})
		return
	}

	var accessToken string

	// TODO: change this when implementing refresh token
	var _ string

	accessToken, _, err = GenerateStandardToken(cpskUser.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to generate access token: %s", err.Error()),
		})
		return
	}

	c.JSON(respStatus, gin.H{
		"user":         cpskUser,
		"access_token": accessToken,
	})
	// Return user that got query from database or newly created one
}

// CompanyGoogleLoginHandler handles Google login authentication for company role, exchanges code for user
// info, checks and creates user in the database, generates an access token, and returns user
// information with the access token.
func (h *OauthLoginHandler) CompanyGoogleLoginHandler(c *gin.Context) {

	uInfo, err := h.getUserInfo(c)
	if err != nil {
		return
	}

	respStatus := http.StatusOK

	// Check does user are already in DB or not
	var user model.User
	var companyUser model.Company
	err = h.DB.Where("google_id = ?", uInfo.GID).First(&user).Error

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):

		verified := model.StatusPending
		if strings.ToLower(strings.TrimSpace(os.Getenv("BYPASS_VERIFICATION"))) == "true" {
			verified = model.StatusVerified
		}

		companyUser = model.Company{
			User: model.User{
				Email:          &uInfo.Email,
				GoogleID:       uInfo.GID,
				Username:       uInfo.FirstName,
				Role:           model.RoleCompany,
				ProfilePicture: uInfo.ProfilePicture,
			},
			VerifiedStatus: verified,
		}

		if err := h.DB.Create(&companyUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to create user: %s", err.Error()),
			})
			return
		}

		respStatus = http.StatusCreated

	case err == nil:

		if err := h.DB.Preload("User").Where("user_id = ?", user.ID).First(&companyUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to retrieve user data: %s", err.Error()),
			})
			return
		}

	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}

	var accessToken string

	// TODO: change this when implementing refresh token
	var _ string

	accessToken, _, err = GenerateStandardToken(companyUser.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to generate access token: %s", err.Error()),
		})
		return
	}

	c.JSON(respStatus, gin.H{
		"user":         companyUser,
		"access_token": accessToken,
	})
	// Return user that got query from database or newly created one
}

// Callback function in Go retrieves a query parameter named "code" from the request and returns it
// in a JSON response.
func (h *OauthLoginHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	c.JSON(http.StatusOK, gin.H{
		"code": code,
	})
}
