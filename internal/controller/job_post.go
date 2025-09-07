package controller

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/util"
	"fmt"
	"net/http"
	// "time"

	"github.com/gin-gonic/gin"
)

func CreateJobPostHandler(c *gin.Context) {
	// Get user
	user := util.ExtractUser(c)

	// construct job post from request
	var jobPost model.JobPost
	c.ShouldBindJSON(&jobPost)

	// save job post
	jobPost.CompanyID = user.ID
	if err := database.DBinstance.Create(&jobPost); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprint("Failed to create job post: ", err),
		})
		return
	}

	// response
	c.JSON(http.StatusCreated, jobPost)
}

// func GetAllPost(c *gin.Context) {
// 	database.DBinstance.Where("expiring > ?", time.Now()).Order()
// }