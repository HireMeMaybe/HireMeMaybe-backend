package controller

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UploadResume function handles the process of uploading a resume file for a user and updating the
// user's information in the database.
func UploadResume(c *gin.Context) {
	var cpskUser = model.CPSKUser{}

	user := utilities.ExtractUser(c)

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

// companyUpload function handles process of reading files from company upload.
func companyUpload(c *gin.Context, fName string) (model.Company, []byte) {
	var company = model.Company{}

	u, _ := c.Get("user")
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User information not provided",
		})
		return company, nil
	}

	user, ok := u.(model.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to assert type",
		})
		return company, nil
	}

	// Retrieve original profile from DB
	if err := database.DBinstance.Preload("User").Where("user_id = ?", user.ID.String()).First(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to retrieve user information from database: %s", err.Error()),
		})
		return company, nil
	}

	rawFile, err := c.FormFile(fName)
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": err.Error(),
		})
		return company, nil
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to retrieve file: %s", err.Error()),
		})
		return company, nil
	}

	allowedExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}
	extension := strings.ToLower(filepath.Ext(rawFile.Filename))

	if !allowedExtensions[extension] {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{
			"error": fmt.Sprintf("Unsupported file extension: %s", extension),
		})
		return company, nil
	}

	f, err := rawFile.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot open file"})
		return company, nil
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal("Failed to close file")
		}
	}()

	fileBytes, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot read file"})
		return company, nil
	}

	return company, fileBytes
}

// UploadLogo function handles company's logo uploading and updating company profile in database.
func UploadLogo(c *gin.Context) {

	company, fileBytes := companyUpload(c, "logo")

	if fileBytes == nil {
		return
	}

	company.Logo.Content = fileBytes
	company.Logo.Extension = "jpg"

	if err := database.DBinstance.Session(&gorm.Session{FullSaveAssociations: true}).Save(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to update user information: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, company)
}

// UploadBanner function handles company's banner uploading and updating company profile in database.
func UploadBanner(c *gin.Context) {
	company, fileBytes := companyUpload(c, "banner")

	if fileBytes == nil {
		return
	}

	company.Banner.Content = fileBytes
	company.Banner.Extension = "jpg"

	if err := database.DBinstance.Session(&gorm.Session{FullSaveAssociations: true}).Save(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to update user information: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, company)
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
