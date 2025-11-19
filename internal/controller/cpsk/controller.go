// Package cpsk provides HTTP handlers for CPSK-related operations.
package cpsk

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CPSKController handles CPSK related endpoints
type CPSKController struct {
	DB *database.DBinstanceStruct
}

// NewCPSKController creates a new instance of CPSKController
func NewCPSKController(db *database.DBinstanceStruct) *CPSKController {
	return &CPSKController{
		DB: db,
	}
}

type editCPSKUser struct {
	model.EditableCPSKInfo
	model.EditableUserInfo
}

// EditCPSKProfile in Go handles editing a user's profile information, including
// retrieving the original profile from the database, updating the information, and saving the changes.
// @Summary Edit CPSK profile
// @Description Overwrite CPSK profile and save into database
// @Description Sensitive field like id, file, and application can't be overwritten
// @Tags CPSK
// @Accept json
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param cpsk_profile body editCPSKUser true "CPSK info to be written"
// @Success 200 {object} model.CPSKUser "Successfully overwrite"
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header or request body"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Not logged in as CPSK, User is banned"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /cpsk/profile [patch]
func (jc *CPSKController) EditCPSKProfile(c *gin.Context) {

	var cpskUser = model.CPSKUser{}

	user, err := utilities.ExtractUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, utilities.ErrorResponse{Error: err.Error()})
		return
	}

	// Retrieve original profile from DB
	if err := jc.DB.Preload("User").Where("user_id = ?", user.ID.String()).First(&cpskUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to retrieve user information from database: %s", err.Error()),
		})
		return
	}

	edited := editCPSKUser{}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&edited); err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: fmt.Sprintf("Invalid request body: %s", err.Error()),
		})
		return
	}

	utilities.MergeNonEmpty(&cpskUser.User.EditableUserInfo, &edited.EditableUserInfo)
	utilities.MergeNonEmpty(&cpskUser.EditableCPSKInfo, &edited.EditableCPSKInfo)

	if err := jc.DB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&cpskUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to update user information: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, cpskUser)
}

// GetMyCPSKProfile retrieves a user's CPSK profile from the database and returns it as
// a JSON response.
// @Summary Retrieve CPSK profile from database
// @Tags CPSK
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Success 200 {object} model.CPSKUser "Successfully retrieve CPSK profile"
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Not logged in as CPSK, User is banned"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /cpsk/myprofile [get]
func (jc *CPSKController) GetMyCPSKProfile(c *gin.Context) {
	user, err := utilities.ExtractUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, utilities.ErrorResponse{Error: err.Error()})
		return
	}

	cpskUser := model.CPSKUser{}

	// Retrieve original profile from DB
	if err := jc.DB.Preload("User").
		Preload("Resume").
		Preload("Applications").
		Preload("Applications.Answer").
		Where("user_id = ?", user.ID.String()).First(&cpskUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to retrieve user information from database: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, cpskUser)
}
