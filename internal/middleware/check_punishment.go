package middleware

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// CheckPunishment check whether user is punished or not
func CheckPunishment(db *database.DBinstanceStruct, punishmentType string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, err := utilities.ExtractUser(ctx)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, utilities.ErrorResponse{Error: err.Error()})
			return
		}

		// Of course, Admin can't be punished
		if user.Role == model.RoleAdmin {
			ctx.Next()
			return
		}

		if user.Punishment == nil {
			ctx.Next()
			return
		}

		if user.Punishment.PunishmentType != punishmentType {
			ctx.Next()
			return
		}

		if user.Punishment.PunishEnd == nil {
			ctx.AbortWithStatusJSON(http.StatusForbidden, utilities.ErrorResponse{
				Error: "You don't have access to this endpoint due to permanent punishment",
			})
			return
		}

		if !time.Now().After(*user.Punishment.PunishEnd) {
			ctx.AbortWithStatusJSON(http.StatusForbidden, utilities.ErrorResponse{
				Error: fmt.Sprintf("You don't have access to this endpoint until: %s", *user.Punishment.PunishEnd),
			})
			return
		}

		punishment := model.PunishmentStruct{}
		punishmentID := user.PunishmentID
		user.Punishment = nil
		user.PunishmentID = nil

		if err := db.Save(&user).Error; err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to update user information: %s", err.Error()),
			})
			return
		}

		if punishmentID != nil {
			if err := db.Where("id = ?", punishmentID).Delete(&punishment).Error; err != nil {
				ctx.AbortWithStatusJSON(http.StatusInternalServerError, utilities.ErrorResponse{
					Error: fmt.Sprintf("Failed to delete punishment record: %s", err.Error()),
				})
				return
			}
		}

		ctx.Next()
	}
}
