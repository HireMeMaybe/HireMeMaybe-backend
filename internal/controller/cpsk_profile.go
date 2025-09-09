// Package controller contains handler for several endpoints
package controller

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/util"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// EditCPSKProfile in Go handles editing a user's profile information, including
// retrieving the original profile from the database, updating the information, and saving the changes.
func EditCPSKProfile(c *gin.Context) {

	var cpskUser = model.CPSKUser{}

	user := util.ExtractUser(c)

	// Retrieve original profile from DB
	if err := database.DBinstance.Preload("User").Where("user_id = ?", user.ID.String()).First(&cpskUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to retrieve user information from database: %s", err.Error()),
		})
		return
	}
	// Save resumeID
	resumeID := cpskUser.ResumeID

	if err := c.ShouldBindJSON(&cpskUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to retrieve user information: %s", err.Error()),
		})
		return
	}
	// Put saved resumeID to prevent resumeID changing
	cpskUser.ResumeID = resumeID

	if err := database.DBinstance.Session(&gorm.Session{FullSaveAssociations: true}).Save(&cpskUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to update user information: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, cpskUser)
}

// GetMyCPSKProfile retrieves a user's CPSK profile from the database and returns it as
// a JSON response.
func GetMyCPSKProfile(c *gin.Context) {
	user := util.ExtractUser(c)

	cpskUser := model.CPSKUser{}

	// Retrieve original profile from DB
	if err := database.DBinstance.Preload("User").Preload("Resume").Where("user_id = ?", user.ID.String()).First(&cpskUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to retrieve user information from database: %s", err.Error()),
		})
	}

	c.JSON(http.StatusOK, cpskUser)
}
