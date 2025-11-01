package company

import "HireMeMaybe-backend/internal/database"

import (
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)


type CompanyController struct {
	DB *database.DBinstanceStruct
}

func NewCompanyController(db *database.DBinstanceStruct) *CompanyController {
	return &CompanyController{
		DB: db,
	}
}

type editCompanyUser struct {
	model.EditableCompanyInfo
	model.EditableUserInfo
}

// GetCompanyProfile function retrieve company profile from database
// and response as JSON format.
// @Summary Retrieve company profile from database
// @Tags Company
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Success 200 {object} model.CompanyUser "Successfully retrieve company profile"
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Not logged in as company, User is banned"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /company/myprofile [get]
func (jc *CompanyController) GetCompanyProfile(c *gin.Context) {
	user := utilities.ExtractUser(c)

	company := model.CompanyUser{}

	// Retrieve company profile from database.
	if err := jc.DB.Preload("User").
		Preload("Logo").
		Preload("Banner").
		Preload("JobPost").
		Where("user_id = ?", user.ID.String()).
		First(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to retrieve user information from database: %s", err.Error()),
		})
	}

	c.JSON(http.StatusOK, company)
}

// EditCompanyProfile function overwrite company profile, save into database
// ,and response edited profile as JSON format.
// @Summary Edit company profile
// @Description Overwrite company profile and save into database
// @Description Sensitive field like id, file, verified status, and job post can't be overwritten
// @Tags Company
// @Accept json
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param company_profile body editCompanyUser true "Company info to be written"
// @Success 200 {object} model.CompanyUser "Successfully overwrite"
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header or request body"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Not logged in as company, User is banned"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /company/profile [patch]
func (jc *CompanyController) EditCompanyProfile(c *gin.Context) {
	user := utilities.ExtractUser(c)

	company := model.CompanyUser{}

	// Retrieve company profile from database
	if err := jc.DB.
		Preload("User").
		Where("user_id = ?", user.ID.String()).
		First(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Fail to retrieve user information from database: %s", err.Error()),
		})
		return
	}

	edited := editCompanyUser{}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&edited); err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: fmt.Sprintf("Invalid request body: %s", err.Error()),
		})
		return
	}

	utilities.MergeNonEmpty(&company.EditableCompanyInfo, &edited.EditableCompanyInfo)
	utilities.MergeNonEmpty(&company.User.EditableUserInfo, &edited.EditableUserInfo)

	// Save updated profile to database
	if err := jc.DB.Session(&gorm.Session{FullSaveAssociations: true}).
		Save(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to update user information: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, company)
}

// GetCompanyByID retrieves a company by its user ID (company_id) and preloads JobPost, Logo, Banner and User.
// @Summary Retrieve company profile from database by given ID
// @Tags Company
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param company_id path string true "ID of company"
// @Success 200 {object} model.CompanyUser "Successfully retrieve company profile"
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "User is banned"
// @Failure 404 {object} utilities.ErrorResponse "Company not exist"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /company/profile/{company_id} [get]
// @Router /company/{company_id} [get]
func (jc *CompanyController) GetCompanyByID(c *gin.Context) {
	companyID := c.Param("company_id")

	company := model.CompanyUser{}

	// Retrieve company profile from database with JobPost preloaded.
	if err := jc.DB.Preload("User").
		Preload("Logo").
		Preload("Banner").
		Preload("JobPost").
		Where("user_id = ?", companyID).
		First(&company).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, utilities.ErrorResponse{Error: "Company not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to retrieve company information from database: %s", err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, company)
}
