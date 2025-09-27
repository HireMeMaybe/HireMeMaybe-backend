package controller

import (
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// ApplicationHandler handles the creation of a new job application by a CPSK user.
func (j *JobController) ApplicationHandler(c *gin.Context) {
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
	if err := j.DB.
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
	if err := j.DB.Create(&application).Error; err != nil {
		var pqErr *pgconn.PgError
		// If the error is a foreign key violation, mean PostID or ResumeID is invalid
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23503" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid PostID or ResumeID: %s", err.Error())})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create application: %s", err.Error())})
		return
	}

	// Return response
	c.JSON(http.StatusCreated, application)
}
