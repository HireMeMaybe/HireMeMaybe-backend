package admin

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminController struct {
	DB *database.DBinstanceStruct
}

func NewAdminController(db *database.DBinstanceStruct) *AdminController {
	return &AdminController{
		DB: db,
	}
}

// GetCompanies function query the result from the database based on given query "verify" and "punishment"
// @Summary Get companies based on given query
// @Description Only admin can access this endpoints
// @Description If no query given, the server will return all companies
// @Tags Admin
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param verify query string false "Only pending, unverified, or verified with case insensitive" example(pending unverified)
// @Param punishment query string false "Only ban, or suspend with case insensitive" example(ban suspend)
// @Success 200 {array} model.CompanyUser
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Do not logged in as admin"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /get-companies [get]
func (jc *AdminController) GetCompanies(c *gin.Context) {
	rawVerify := c.Query("verify")
	fmt.Println(rawVerify)
	rawPunishment := c.Query("punishment")

	result := jc.DB.Preload("User").Preload("User.Punishment").Preload("JobPost")
	if rawVerify != "" {
		verify := strings.Split(rawVerify, " ")
		for i := range verify {
			verify[i] = strings.ToUpper(verify[i][:1]) + strings.ToLower(verify[i][1:])
		}
		result = result.Where("verified_status IN ?", verify)
	}

	if rawPunishment != "" {
		punishment := strings.Split(rawPunishment, " ")
		for i := range punishment {
			punishment[i] = strings.ToLower(punishment[i])
		}
		result = result.Joins("JOIN users ON users.id = company_users.user_id").
			Joins("JOIN punishment_structs ON punishment_structs.id = users.punishment_id").
			Where("punishment_type IN ?", punishment).
			Where("(punish_end > ? OR punish_end IS NULL)", time.Now())
	}

	var companyUser []model.CompanyUser

	result = result.Find(&companyUser)

	if err := result.Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, companyUser)
}

// GetCPSK function query the result from the database based on given query "punishment"
// @Summary Get CPSK based on given query
// @Description Only admin can access this endpoints
// @Description If no query given, the server will return all CPSK
// @Tags Admin
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param punishment query string false "Only ban, or suspend with case insensitive" example(ban suspend)
// @Success 200 {array} model.CPSKUser
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Do not logged in as admin"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /get-cpsk [get]
func (jc *AdminController) GetCPSK(c *gin.Context) {
	rawPunishment := c.Query("punishment")
	result := jc.DB.Preload("User").Preload("User.Punishment")
	if rawPunishment != "" {
		punishment := strings.Split(rawPunishment, " ")
		for i := range punishment {
			punishment[i] = strings.ToLower(punishment[i])
		}
		result = result.Joins("JOIN users ON users.id = cpsk_users.user_id").
			Joins("JOIN punishment_structs ON punishment_structs.id = users.punishment_id").
			Where("punishment_type IN ?", punishment).
			Where("(punish_end > ? OR punish_end IS NULL)", time.Now())
	}

	var cpskUser []model.CPSKUser

	result = result.Find(&cpskUser)

	if err := result.Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, cpskUser)
}

// GetVisitors function query the result from the database based on given query "punishment"
// @Summary Get visitors based on given query
// @Description Only admin can access this endpoints
// @Description If no query given, the server will return all visitors
// @Tags Admin
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param punishment query string false "Only ban, or suspend with case insensitive" example(ban suspend)
// @Success 200 {array} model.VisitorUser
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Do not logged in as admin"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /get-visitors [get]
func (jc *AdminController) GetVisitors(c *gin.Context) {
	rawPunishment := c.Query("punishment")
	result := jc.DB.Preload("User").Preload("User.Punishment")
	if rawPunishment != "" {
		punishment := strings.Split(rawPunishment, " ")
		for i := range punishment {
			punishment[i] = strings.ToLower(punishment[i])
		}
		result = result.Joins("JOIN users ON users.id = visitor_users.user_id").
			Joins("JOIN punishment_structs ON punishment_structs.id = users.punishment_id").
			Where("punishment_type IN ?", punishment).
			Where("(punish_end > ? OR punish_end IS NULL)", time.Now())
	}

	var visitorUser []model.VisitorUser

	result = result.Find(&visitorUser)

	if err := result.Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, visitorUser)
}

// VerifyCompany function allow admin to change status of given company id to Verified or Unverified
// @Summary Verify, or unverify companies
// @Description Only admin can access this endpoints
// @Tags Admin
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param company_id path string true "Company ID"
// @Param status query string false "Status is case insensitive and allow only unverified, or verified (verified by default)" default(verified)
// @Success 200 {object} model.CompanyUser
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header, or Invalid request body"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Do not logged in as admin"
// @Failure 404 {object} utilities.ErrorResponse "Given company ID not found"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /verify-company/{company_id} [patch]
func (jc *AdminController) VerifyCompany(c *gin.Context) {
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
		return

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
