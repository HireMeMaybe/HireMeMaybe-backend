package controller

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/util"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UploadResume function handles the process of uploading a resume file for a user and updating the
// user's information in the database.
func UploadResume(c *gin.Context) {
	var cpskUser = model.CPSKUser{}

	user := util.ExtractUser(c)

	// Retrieve original profile from DB
	if err := database.DBinstance.Preload("User").Where("user_id = ?", user.ID.String()).First(&cpskUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to retrieve user information from database: %s", err.Error()),
		})
		return
	}

	rawFile, err := c.FormFile("resume")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to retrieve file: %s", err.Error()),
		})
		return
	}

	f, err := rawFile.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot open file"})
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal("Failed to close file")
		}
	}()

	fileBytes, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot read file"})
		return
	}

	cpskUser.Resume.Content = fileBytes
	cpskUser.Resume.Extension = "pdf"

	if err := database.DBinstance.Session(&gorm.Session{FullSaveAssociations: true}).Save(&cpskUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to update user information: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, cpskUser)
}

// GetFile function retrieves a file from the database and sends it as a downloadable attachment in
// the response.
func GetFile(c *gin.Context) {
	var file model.File
	id := c.Param("id")

	if err := database.DBinstance.First(&file, id).Error; err != nil {
		c.String(http.StatusNotFound, "File not found")
		return
	}

	// Set Content-Disposition with file name and extension
	c.Writer.Header().Set("Content-Disposition", "attachment; filename="+fmt.Sprint(file.ID)+"."+file.Extension)
	c.Writer.Header().Set("Content-Type", "application/octet-stream")
	c.Writer.Header().Set("Content-Length", fmt.Sprint(len(file.Content)))

	// Write file data to response
	_, err := c.Writer.Write(file.Content)
	if err != nil {
		if !c.Writer.Written() {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to send file content",
			})
		} else {
			c.Abort()
		}
		return
	}
}
