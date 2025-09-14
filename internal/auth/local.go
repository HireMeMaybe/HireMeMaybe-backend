package auth

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// LocalRegisterHandler function handles local registration by receiving username and password
// do nothing if username already exist in the database
// do nothing if password is shorter than 8 characters
func LocalRegisterHandler(c *gin.Context) {
	var info struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role" binding:"required,oneof=cpsk company"`
	}

	if err := c.ShouldBindJSON(&info); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Username, password, and Role (Only 'cpsk' or 'company) must be provided",
		})
		return
	}

	var user model.User
	err := database.DBinstance.Where("username = ?", info.Username).First(&user).Error

	switch {
	case err == nil:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Username already exist",
		})
		return

	case errors.Is(err, gorm.ErrRecordNotFound):
		// Do nothing

	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}

	if len(info.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Password should longer or equal to 8 characters",
		})
		return
	}

	hashedPassword, err := utilities.HashPassword(info.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed hash password: %s", err.Error()),
		})
		return
	}

	switch info.Role {
	case "cpsk":
		cpskUser := model.CPSKUser{
			User: model.User{
				Username: info.Username,
				Password: hashedPassword,
				Role:     model.RoleCPSK,
			},
		}
		if err := database.DBinstance.Create(&cpskUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to create user: %s", err.Error()),
			})
			return
		}

		accessToken, _, err := generateToken(cpskUser.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to generate access token: %s", err.Error()),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"user":         cpskUser,
			"access_token": accessToken,
		})
	case "company":
		verified := model.StatusPending
		if strings.ToLower(strings.TrimSpace(os.Getenv("BYPASS_VERIFICATION"))) == "true" {
			verified = model.StatusVerified
		}

		companyUser := model.Company{
			User: model.User{
				Username: info.Username,
				Password: hashedPassword,
				Role:     model.RoleCompany,
			},
			VerifiedStatus: verified,
		}
		if err := database.DBinstance.Create(&companyUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to create user: %s", err.Error()),
			})
			return
		}

		accessToken, _, err := generateToken(companyUser.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to generate access token: %s", err.Error()),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"user":         companyUser,
			"access_token": accessToken,
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Role '%s' not allowed", info.Role),
		})
	}
}

// LocalLoginHandler function handles local login by receiving username and password
// do nothing if username does not exist in the database
// do nothing if password is incorrect
func LocalLoginHandler(c *gin.Context) {
	var info struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&info); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Username or password is not provided",
		})
		return
	}

	var user model.User
	err := database.DBinstance.Where("username = ?", info.Username).First(&user).Error

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Username or password is incorrect",
		})
		return

	case err == nil:
		// Do nothing

	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}

	if user.Password == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Username or password is incorrect",
		})
		return
	}

	if !utilities.VerifyPassword(info.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Username or password is incorrect",
		})
		return
	}

	switch user.Role {
	case model.RoleCPSK:
		var cpskUser model.CPSKUser
		if err := database.DBinstance.Preload("User").Where("user_id = ?", user.ID).First(&cpskUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to retrieve user data: %s", err.Error()),
			})
			return
		}

		accessToken, _, err := generateToken(cpskUser.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to generate access token: %s", err.Error()),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user":         cpskUser,
			"access_token": accessToken,
		})
	case model.RoleCompany:
		var companyUser model.Company
		if err := database.DBinstance.Preload("User").Where("user_id = ?", user.ID).First(&companyUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to retrieve user data: %s", err.Error()),
			})
			return
		}

		accessToken, _, err := generateToken(companyUser.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to generate access token: %s", err.Error()),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user":         companyUser,
			"access_token": accessToken,
		})
	default:
		accessToken, _, err := generateToken(user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to generate access token: %s", err.Error()),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user":         user,
			"access_token": accessToken,
		})
	}
}
