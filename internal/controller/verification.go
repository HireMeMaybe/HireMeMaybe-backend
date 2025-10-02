package controller

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type verificationInfo struct {
	CompanyID string `json:"company_id" binding:"required"`
	Status    string `json:"status" binding:"required"`
}

// GetCompanies function query the result from the database based on given query "status"
// which mean VerifiedStatus is the condition for the query
// @Summary Get companies based on given status
// @Description Only admin can access this endpoints
// @Description If no query given, the server will return all companies
// @Tags Admin
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param status query string false "Only pending, unverified, or verified with case insensitive" example(pending+unverified)
// @Success 200 {array} model.Company
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Do not logged in as admin"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /get-companies [get]
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
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, companyUser)
}

// VerifyCompany function allow admin to change status of given company id to Verified or Unverified
// @Summary Verify, or unverify companies
// @Description Only admin can access this endpoints
// @Tags Admin
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param Info body verificationInfo true "Company ID and status with only case insensitive unverified, or verified"
// @Success 200 {object} model.Company
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header, or Invalid request body"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Do not logged in as admin"
// @Failure 404 {object} utilities.ErrorResponse "Given company ID not found"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /verify-company [put]
func VerifyCompany(c *gin.Context) {
	var info verificationInfo

	if err := c.ShouldBindJSON(&info); err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: "CompanyID and Status must be provided",
		})
		return
	}

	info.Status = strings.ToUpper(info.Status[:1]) + strings.ToLower(info.Status[1:])
	allowedStatus := map[string]bool{
		model.StatusVerified:   true,
		model.StatusUnverified: true,
	}

	if !allowedStatus[info.Status] {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: fmt.Sprintf("Unknown status: %s", info.Status),
		})
		return
	}

	var company model.Company
	err := database.DBinstance.Preload("User").Where("user_id = ?", info.CompanyID).First(&company).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, utilities.ErrorResponse{
			Error: fmt.Sprintf("%s does not exist in the database", info.CompanyID),
		})

	case err == nil:
		// Do nothing

	default:
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}
	company.VerifiedStatus = info.Status

	if err := database.DBinstance.Session(&gorm.Session{FullSaveAssociations: true}).
		Save(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to update user information: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, company)
}
