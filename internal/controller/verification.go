package controller

import (
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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
func (jc *JobController) GetCompanies(c *gin.Context) {
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

	var companyUser []model.CompanyUser

	err := jc.DB.
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
// @Param company_id path string true "Company ID"
// @Param status query string false "Status is case insensitive and allow only unverified, or verified (verified by default)" default(verified)
// @Success 200 {object} model.Company
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header, or Invalid request body"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Do not logged in as admin"
// @Failure 404 {object} utilities.ErrorResponse "Given company ID not found"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /verify-company/{company_id} [patch]
func (jc *JobController) VerifyCompany(c *gin.Context) {
	companyID := c.Param("company_id")
	status := c.Query("status")

	if status == "" {
		status = "verified"
	}

	status = strings.ToUpper(status[:1]) + strings.ToLower(status[1:])
	allowedStatus := map[string]bool{
		model.StatusVerified:   true,
		model.StatusUnverified: true,
	}

	if !allowedStatus[status] {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: fmt.Sprintf("Unknown status: %s", status),
		})
		return
	}

	var company model.CompanyUser
	err := jc.DB.Preload("User").Where("user_id = ?", companyID).First(&company).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, utilities.ErrorResponse{
			Error: fmt.Sprintf("%s does not exist in the database", companyID),
		})

	case err == nil:
		// Do nothing

	default:
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}
	company.VerifiedStatus = status

	if err := jc.DB.Session(&gorm.Session{FullSaveAssociations: true}).
		Save(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to update user information: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, company)
}

type aiVerificationResponse struct {
	Company    model.CompanyUser `json:"company"`
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
	var company model.CompanyUser
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
