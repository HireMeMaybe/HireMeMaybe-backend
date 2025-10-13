package middleware

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CheckPunishment check whether user is punished or not
func CheckPunishment(db *database.DBinstanceStruct, punishmentType string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := utilities.ExtractUser(ctx)

		if ctx.IsAborted() {
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

		user.Punishment = nil
		if err := db.Session(&gorm.Session{FullSaveAssociations: true}).
			Save(&user).Error; err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to update user information: %s", err.Error()),
			})
			return
		}

		ctx.Next()
	}
}
