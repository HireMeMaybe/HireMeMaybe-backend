package middleware

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CheckPunishment check whether user is punished or not
func CheckPunishment(db *database.DBinstanceStruct, punishmentType string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, err := utilities.ExtractUser(ctx)

		// Of course, Admin can't be punished
		if user.Role == model.RoleAdmin {
			ctx.Next()
			return
		}

		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, utilities.ErrorResponse{Error: err.Error()})
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

		if msg, status, err := database.RemovePunishment(user, db); err != nil {
			ctx.AbortWithStatusJSON(status, utilities.ErrorResponse{
				Error: msg,
			})
			return
		}

		ctx.Next()
	}
}
