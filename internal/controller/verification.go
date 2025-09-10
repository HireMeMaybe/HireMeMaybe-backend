package controller

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetCompanies function query the result from the database based on given query "status"
// which mean VerifiedStatus is the condition for the query
func GetCompanies(c *gin.Context) {
	rawQ := c.Query("status")
	var q []string
	if rawQ == "" {
		q = []string{model.StatusPending, model.StatusUnverified, model.StatusVerified}
	} else {
		q = strings.Split(rawQ, " ")
		for i := range q {
			q[i] = strings.ToUpper(q[i][:1]) + strings.ToLower(q[i][1:])
		}
	}

	var companyUser []model.Company

	err := database.DBinstance.
		Preload("User").
		Where("verified_status IN ?", q).
		Find(&companyUser).
		Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": companyUser})
}

func VerifyCompany(c *gin.Context) {
	var info struct {
		CompanyID string `json:"company_id" binding:"required"`
		Status    string `json:"status" binding:"required,oneof=Verified Unverified"`
	}

	if err := c.ShouldBindJSON(&info); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "CompanyID and Status (Only Verified or Unverified) must be provided",
		})
		return
	}

	var company model.Company
	err := database.DBinstance.Preload("User").Where("user_id = ?", info.CompanyID).First(&company).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("%s does not exist in the database", info.CompanyID),
		})

	case err == nil:
		// Do nothing

	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}
	company.VerifiedStatus = info.Status

	if err := database.DBinstance.Session(&gorm.Session{FullSaveAssociations: true}).
		Save(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to update user information: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, company)
}
