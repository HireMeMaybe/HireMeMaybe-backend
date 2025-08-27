package controller

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// The function `EditCPSKProfile` in Go handles editing a user's profile information, including
// retrieving the original profile from the database, updating the information, and saving the changes.
func EditCPSKProfile(c *gin.Context) {
	var cpskUser = model.CPSKUser{}

	u, _ := c.Get("user")
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User information not provided",
		})
		return
	}

	user, ok := u.(model.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to assert type",
		})
		return
	}

	// Retrieve original profile from DB
	if err := database.DBinstance.Preload("User").Where("user_id = ?", user.ID.String()).First(&cpskUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to retrieve user information from database: %s", err.Error()),
		})
		return
	}

	if err := c.ShouldBindJSON(&cpskUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to retrieve user information: %s", err.Error()),
		})
		return
	}

	// cpskUser.UserID = user.ID

	// var dummy struct {
	// 	Tel string `json:"tel"`
	// }

	// c.GetRawData()

	// if err := c.ShouldBindJSON(&dummy); err != nil {
	// 	println("I'm here 1")
	// 	if err != io.EOF {
	// 		c.JSON(http.StatusInternalServerError, gin.H{
	// 			"error": fmt.Sprintf("Failed to retrieve user information: %s", err.Error()),
	// 		})
	// 		return
	// 	}
	// } else if dummy.Tel != ""{
	// 	println("I'm here 2")
	// 	cpskUser.User.Tel = &dummy.Tel	
	// }
	if err := database.DBinstance.Session(&gorm.Session{FullSaveAssociations: true}).Save(&cpskUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to update user information: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, cpskUser)
}

func GetMyCPSKProfile(c *gin.Context) {
	u, _ := c.Get("user")
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User information not provided",
		})
		return
	}

	user, ok := u.(model.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to assert type",
		})
		return
	}

	cpskUser := model.CPSKUser{}
	
	// Retrieve original profile from DB
	if err := database.DBinstance.Preload("User").Where("user_id = ?", user.ID.String()).First(&cpskUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to retrieve user information from database: %s", err.Error()),
		})
	}

	c.JSON(http.StatusOK, cpskUser)
}