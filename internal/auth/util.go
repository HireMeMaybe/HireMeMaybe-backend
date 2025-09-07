package auth

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func LoginOrRegister[userModel interface{}](user *userModel, gid string, c *gin.Context) (int) {
	
	baseUser := model.User{}
	err := database.DBinstance.Where("google_id = ?", gid).First(&baseUser).Error
	respStatus := http.StatusOK

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):

		if err := database.DBinstance.Create(&user).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to create user: %v", err.Error()),
			})
		}

		respStatus = http.StatusCreated
	case err == nil:
		if err := database.DBinstance.Preload("User").Where("user_id = ?", baseUser.ID).First(&user).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to retrieve user data: %v", err.Error()),
			})
		}
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Database error: %v", err.Error()),
		})
	}

	return respStatus
}