package controller

import (
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type aiVerificationResponse struct {
	Company    model.Company `json:"company"`
	AIDecision string        `json:"ai_decision"`
	Reasoning  string        `json:"reasoning"`
	Confidence string        `json:"confidence"`
}

// AIVerifyCompany uses AI to analyze company information and automatically verify or reject
// @Summary Use AI to verify your company
// @Description Company can request AI verification of their own profile. AI analyzes company data and makes verification decision
// @Tags Company
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Success 200 {object} aiVerificationResponse
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Not logged in as company"
// @Failure 404 {object} utilities.ErrorResponse "Company profile not found"
// @Failure 500 {object} utilities.ErrorResponse "Database error or AI service error"
// @Router /company/ai-verify [post]
func (jc *JobController) AIVerifyCompany(c *gin.Context) {
	// Extract user from token (middleware already validated it's a company)
	user := utilities.ExtractUser(c)

	// Fetch company information with all necessary preloads
	var company model.Company
	err := jc.DB.
		Preload("User").
		Preload("Logo").
		Preload("Banner").
		Preload("JobPost").
		Where("user_id = ?", user.ID.String()).
		First(&company).Error

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, utilities.ErrorResponse{
			Error: "Company profile not found",
		})
		return

	case err != nil:
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}

	// Call AI service to analyze company (local function in controller package)
	result, err := VerifyCompanyWithAI(company)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("AI verification failed: %s", err.Error()),
		})
		return
	}

	// Determine new status based on AI decision
	var newStatus string
	if result.ShouldVerify {
		newStatus = model.StatusVerified
	} else {
		newStatus = model.StatusUnverified
	}

	// Update company verification status
	company.VerifiedStatus = newStatus

	if err := jc.DB.Session(&gorm.Session{FullSaveAssociations: true}).
		Save(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to update company verification status: %s", err.Error()),
		})
		return
	}

	// Prepare response
	response := aiVerificationResponse{
		Company:    company,
		AIDecision: newStatus,
		Reasoning:  result.Reasoning,
		Confidence: result.Confidence,
	}

	c.JSON(http.StatusOK, response)
}
