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

// GetCompanyProfile function retrieve company profile from database
// and response as JSON format.
func GetCompanyProfile(c *gin.Context) {
	user := utilities.ExtractUser(c)

	company := model.Company{}

	// Retrieve company profile from database.
	if err := database.DBinstance.Preload("User").
		Preload("Logo").
		Preload("Banner").
		Preload("JobPost").
		Where("user_id = ?", user.ID.String()).
		First(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to retrieve user information from database: %s", err.Error()),
		})
	}

	c.JSON(http.StatusOK, company)
}

// EditCompanyProfile function overide company profile, save into database
// ,and response edited profile as JSON format.
func EditCompanyProfile(c *gin.Context) {
	user := utilities.ExtractUser(c)

	company := model.Company{}

	// Retrieve company profile from database
	if err := database.DBinstance.
		Preload("User").
		Where("user_id = ?", user.ID.String()).
		First(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Fail to retrieve user information from database: %s", err.Error()),
		})
		return
	}
	// Save unintended to change field
	logoID := company.LogoID
	bannerID := company.BannerID
	status := company.VerifiedStatus

	if err := c.ShouldBindJSON(&company); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to retrieve user information: %s", err.Error()),
		})
		return
	}
	// Put saved unintended field to prevent change
	company.LogoID = logoID
	company.BannerID = bannerID
	company.VerifiedStatus = status

	// Save updated profile to database
	if err := database.DBinstance.Session(&gorm.Session{FullSaveAssociations: true}).
		Save(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to update user information: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, company)
}

// GetCompanyByID retrieves a company by its user ID (company_id) and preloads JobPost, Logo, Banner and User.
func GetCompanyByID(c *gin.Context) {
	companyID := c.Param("company_id")

	company := model.Company{}

	// Retrieve company profile from database with JobPost preloaded.
	if err := database.DBinstance.Preload("User").
		Preload("Logo").
		Preload("Banner").
		Preload("JobPost").
		Where("user_id = ?", companyID).
		First(&company).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Company not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to retrieve company information from database: %s", err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, company)
}
