package auth

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var googleOauth *oauth2.Config

func init() {

	googleOauth = &oauth2.Config{
		ClientID: os.Getenv("CPSK_GOOGLE_AUTH_CLIENT"),
		ClientSecret: os.Getenv("CPSK_GOOGLE_AUTH_SECRET"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
        	"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.openid",
		},
		Endpoint: google.Endpoint,
		RedirectURL:  "http://localhost:8080/auth/google/callback",
	}
}

func GoogleLogin(c *gin.Context) {

	type oauthCode struct {
		Code string `json:"code" binding:"required"`
	}

	var code oauthCode

	// check does body has code
	if err := c.ShouldBindJSON(&code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("No authorization code provided: %s", err.Error()),
		})
		return
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
		return
	}

	client := googleOauth.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to fetch user information: %s", err.Error()),
		})
		return
	}

	type userInfo struct {
		GID 			string `json:"sub"`
		FirstName 	string `json:"given_name"`
		LastName 	string `json:"family_name"`
		Email 		string `json:"email"`
	}

	var uInfo userInfo
	
	err = json.NewDecoder(resp.Body).Decode(&uInfo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to decode user info: %s", err.Error()),
		})
		return
	}
	// Check does user are already in DB or not
	var user model.CPSKUser
	database.DBinstance.Where("google_id = ?",  uInfo.GID).First(&user)
	
	// If user not exist in db create one with provided information
	if user.ID == 0 {
		user = model.CPSKUser{
			GoogleId: uInfo.GID,
			FirstName: uInfo.FirstName,
			LastName: uInfo.LastName,
			ContactInfo: model.ContactInfo{
				Email: &uInfo.Email,
			},
		}
		database.DBinstance.Create(&user)
	}

	c.JSON(http.StatusCreated, &user)
	// If already exist return that user

}

func Callback(c *gin.Context) {
	code := c.Query("code")
	c.JSON(http.StatusOK, gin.H{
		"code": code,
	})
}