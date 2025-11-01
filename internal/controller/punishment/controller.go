package punishment

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PunishmentController struct {
	DB *database.DBinstanceStruct
}

func NewPunishmentController(db *database.DBinstanceStruct) *PunishmentController {
	return &PunishmentController{
		DB: db,
	}
}

// PunishUser handles ban and suspend process for admin
// @Summary Ban or suspend user
// @Description Type of punishment (Only 'ban' or 'suspend' with case insensitive),
// @Description 'at' and 'end' fields must be in 'YYYY-MM-DDTHH:mm:ssZ' format.
// @Description Only 'type' is required 'at' and 'end' are optional
// @Description 'at' will be current time by default
// @Description 'end' leave empty mean permanent punishment
// @Tags Admin
// @Accept json
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param user_id path string true "ID of user to be punished"
// @Param Detail body model.PunishmentStruct true "Detail of punishment"
// @Success 200 {object} utilities.MessageResponse "Successfully punish a user"
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header, request body"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Not logged in as Admin, trying to punish other Admin"
// @Failure 404 {object} utilities.ErrorResponse "User not found"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /punish/{user_id} [put]
func (jc *PunishmentController ) PunishUser(c *gin.Context) {
	userID := c.Param("user_id")

	user := model.User{}
	punishment := model.PunishmentStruct{}
	if err := jc.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, utilities.ErrorResponse{Error: "User not found"})
		return
	}

	if user.Role == model.RoleAdmin {
		c.JSON(http.StatusForbidden, utilities.ErrorResponse{
			Error: "Unable to punish other admin",
		})
		return
	}

	if err := c.ShouldBindJSON(&punishment); err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: fmt.Sprintf("Invalid request body: %s", err.Error()),
		})
		return
	}
	if punishment.PunishAt == nil {
		now := time.Now()
		punishment.PunishAt = &now
	}

	allowedType := []string{model.BanPunishment, model.SuspendPunishment}
	punishment.PunishmentType = strings.ToLower(punishment.PunishmentType)
	if !slices.Contains(allowedType, punishment.PunishmentType) {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: "Invalid request body: type can be only 'ban' or 'suspend'",
		})
		return
	}
	if punishment.PunishEnd != nil {
		if punishment.PunishAt.After(*punishment.PunishEnd) {
			c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
				Error: "Invalid request body: 'end' time must more than 'at' time",
			})
			return
		}
	}

	user.Punishment = &punishment

	if err := jc.DB.Session(&gorm.Session{FullSaveAssociations: true}).
		Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to update user information: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, utilities.MessageResponse{
		Message: fmt.Sprintf("Successfully %s %s", punishment.PunishmentType, user.Username),
	})
}

// DeletePunishmentRecord removes punishment record from user
// @Summary Remove punishment record from user
// @Tags Admin
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param user_id path string true "ID of user to be unpunished"
// @Success 200 {object} utilities.MessageResponse "Successfully remove punishment record from user"
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header, request body"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Not logged in as Admin"
// @Failure 404 {object} utilities.ErrorResponse "User not found"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /punish/{user_id} [delete]
func (jc *PunishmentController ) DeletePunishmentRecord(c *gin.Context) {
	userID := c.Param("user_id")

	user := model.User{}
	if err := jc.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, utilities.ErrorResponse{Error: "User not found"})
		return
	}

	punishmentID := user.PunishmentID
	var punishment model.PunishmentStruct
	user.Punishment = nil
	user.PunishmentID = nil

	if err := jc.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to update user information: %s", err.Error()),
		})
		return
	}

	if punishmentID != nil {
		if err := jc.DB.Where("id = ?", punishmentID).Delete(&punishment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to delete punishment record: %s", err.Error()),
			})
			return
		}
	}

	c.JSON(http.StatusOK, utilities.MessageResponse{
		Message: fmt.Sprintf("Successfully removed punishment record of %s", user.Username),
	})
}
