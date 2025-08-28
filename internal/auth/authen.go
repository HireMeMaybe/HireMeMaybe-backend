package auth

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

var googleOauth *oauth2.Config

func init() {
	googleOauth = &oauth2.Config{
		ClientID:     os.Getenv("CPSK_GOOGLE_AUTH_CLIENT"),
		ClientSecret: os.Getenv("CPSK_GOOGLE_AUTH_SECRET"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.openid",
		},
		Endpoint:    google.Endpoint,
		RedirectURL: "http://localhost:8080/auth/google/callback",
	}
}

func getUserInfo(c *gin.Context) (uInfo struct {
	GID       string `json:"sub"`
	FirstName string `json:"given_name"`
	LastName  string `json:"family_name"`
	Email     string `json:"email"`
}, e error) {

	var code struct {
		Code string `json:"code" binding:"required"`
	}

	// check does body has code
	if err := c.ShouldBindJSON(&code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("No authorization code provided: %s", err.Error()),
		})
		return uInfo, err
	}

	// Exchange code with google and get userinfo
	token, err := googleOauth.Exchange(
		context.Background(),
		code.Code,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to receive token: %s", err.Error()),
		})
		return uInfo, err
	}

	client := googleOauth.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to fetch user information: %s", err.Error()),
		})
		return uInfo, err
	}

	err = json.NewDecoder(resp.Body).Decode(&uInfo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to decode user info: %s", err.Error()),
		})
		return uInfo, err
	}
	return uInfo, nil
}

func CPSKGoogleLoginHandler(c *gin.Context) {

	uInfo, err := getUserInfo(c)
	if err != nil {
		return
	}

	respStatus := http.StatusOK

	// Check does user are already in DB or not
	var user model.User
	var cpskUser model.CPSKUser
	database.DBinstance = database.DBinstance.Debug()
	err = database.DBinstance.Where("google_id = ?", uInfo.GID).First(&user).Error

	// If user not exist in db create one with provided information
	if errors.Is(err, gorm.ErrRecordNotFound) {

		cpskUser = model.CPSKUser{
			User: model.User{
				Email:    &uInfo.Email,
				GoogleId: uInfo.GID,
				Username: uInfo.FirstName,
			},
			FirstName: uInfo.FirstName,
			LastName:  uInfo.LastName,
		}

		if err := database.DBinstance.Create(&cpskUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to create user: %s", err.Error()),
			})
			return
		}

		respStatus = http.StatusCreated

	} else if err == nil {

		if err := database.DBinstance.Preload("User").Where("user_id = ?", user.ID).First(&cpskUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to retrieve user data: %s", err.Error()),
			})
			return
		}

	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}

	var accessToken string

	// TODO: change this when implementing refresh token
	var _ string

	accessToken, _, err = generateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to generate access token: %s", err.Error()),
		})
		return
	}

	c.JSON(respStatus, gin.H{
		"user":        cpskUser,
		"acess_token": accessToken,
	})
	// Return user that got query from database or newly created one
}

func CompanyGoogleLoginHandler(c *gin.Context) {
	uInfo, err := getUserInfo(c)
	if err != nil {
		return
	}

	respStatus := http.StatusOK

	// Check does user are already in DB or not
	var user model.User
	var companyUser model.Company
	database.DBinstance = database.DBinstance.Debug()
	err = database.DBinstance.Where("google_id = ?", uInfo.GID).First(&user).Error

	// If user not exist in db create one with provided information
	if errors.Is(err, gorm.ErrRecordNotFound) {

		companyUser = model.Company{
			User: model.User{
				Email:    &uInfo.Email,
				GoogleId: uInfo.GID,
				Username: uInfo.FirstName,
			},
			VerifiedStatus: "Unverified",
		}

		if err := database.DBinstance.Create(&companyUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to create user: %s", err.Error()),
			})
			return
		}

		respStatus = http.StatusCreated

	} else if err == nil {

		if err := database.DBinstance.Preload("User").Where("user_id = ?", user.ID).First(&companyUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to retrieve user data: %s", err.Error()),
			})
			return
		}

	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}

	var accessToken string

	// TODO: change this when implementing refresh token
	var _ string

	accessToken, _, err = generateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to generate access token: %s", err.Error()),
		})
		return
	}

	c.JSON(respStatus, gin.H{
		"user":        companyUser,
		"acess_token": accessToken,
	})
	// Return user that got query from database or newly created one
}

func Callback(c *gin.Context) {
	code := c.Query("code")
	c.JSON(http.StatusOK, gin.H{
		"code": code,
	})
}
