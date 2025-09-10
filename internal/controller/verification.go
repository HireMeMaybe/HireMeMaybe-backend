package controller

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetCompanies function query the result from the database based on given query "status"
// which mean VerifiedStatus is the condition for the query
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": companyUser})
}
