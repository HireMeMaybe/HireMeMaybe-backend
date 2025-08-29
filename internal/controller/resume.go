package controller

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func UploadResume(c *gin.Context) {
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
	defer f.Close()

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
	c.Writer.Write(file.Content)
}
