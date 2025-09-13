package controller

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func ApplicationHandler(c *gin.Context) {
	// ExtractUser(c)
	user := utilities.ExtractUser(c)

	// Extract application detail from request body
	application := model.Application{}
	if err := c.ShouldBindJSON(&application); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request body: %s", err.Error())})
		return
	}
	
	// Set user CPSKID and Application status to application
	application.CPSKID = user.ID

	// Prevent duplicate applications: check if this CPSK already applied to the same job post
	existing := model.Application{}
	if err := database.DBinstance.
		Where("cpsk_id = ? AND post_id = ?", user.ID, application.PostID).
		First(&existing).Error; err == nil {
		// Found an existing application
		c.JSON(http.StatusBadRequest, gin.H{"error": "You have already applied to this job post"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Some other DB error occurred
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing application"})
		return
	}

	application.Status = model.ApplicationStatusPending

	// Save application to database
	if err := database.DBinstance.Create(&application).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create application"})
		return
	}

	// Return response
	c.JSON(http.StatusCreated, application)
}
