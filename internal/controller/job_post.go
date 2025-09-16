package controller

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
	"gorm.io/gorm/clause"
)

// CreateJobPostHandler handles the creation of a new job post by a company user.
func CreateJobPostHandler(c *gin.Context) {
	// Get user
	user := utilities.ExtractUser(c)

	// Ensure that user is a verified company
	var companyUser model.Company
	if err := database.DBinstance.Where("user_id = ?", user.ID.String()).First(&companyUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only company users can create job posts"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to retrieve company information: %s", err.Error())})
		return
	}
	if companyUser.VerifiedStatus != model.StatusVerified {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only verified companies can create job posts"})
		return
	}

	// construct job post from request
	var jobPost model.JobPost
	if err := c.ShouldBindJSON(&jobPost); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request body: %s", err.Error())})
		return
	}

	// save job post
	jobPost.CompanyID = user.ID
	if err := database.DBinstance.Create(&jobPost).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprint("Failed to create job post: ", err),
		})
		return
	}

	// response
	c.JSON(http.StatusCreated, jobPost)
}

// GetPosts fetches all non-expired job posts from the database and returns them as a JSON response.
func GetPosts(c *gin.Context) {
	rawSearch := c.Query("search")
	rawJobType := c.Query("type")
	rawTag := c.Query("tag")
	rawSalary := c.Query("salary")
	rawCompany := c.Query("company")
	rawIndustry := c.Query("industry")
	rawLocation := c.Query("location")
	rawDesc := c.Query("desc")

	var posts []model.JobPost

	result := database.DBinstance.Where("expiring > ? OR expiring IS NULL", time.Now())

	if rawSearch != "" {
		result = result.Where("title ILIKE ?", "%"+rawSearch+"%")
	}

	if rawJobType != "" {
		result = result.Where("type ILIKE ?", "%"+rawJobType+"%")
	}

	if rawTag != "" {
		result = result.Where("? ILIKE ANY(tags)", rawTag)
	}

	if rawSalary != "" {
		result = result.Where("salary = ?", rawSalary)
	}

	if rawCompany != "" {
		result = result.Preload("Company").Where("name ILIKE ?", "%"+rawCompany+"%")
	}

	if rawIndustry != "" {
		result = result.Preload("Company").Where("industry ILIKE ?", "%"+rawIndustry+"%")
	}

	if rawLocation != "" {
		result = result.Where("location ILIKE ?", "%"+rawLocation+"%")
	}

	result = result.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "post_time"},
		Desc:   strings.ToLower(rawDesc) == "true",
	}).
		Find(&posts)
	if err := result.Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprint("Failed to fetch job post: ", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, posts)
}

// EditJobPost allows a company user to update a job post they own.
func EditJobPost(c *gin.Context) {
	// Use ExtractUser itiesity to get authenticated user
	user := utilities.ExtractUser(c)

	// Get job post id from path
	id := c.Param("id")

	job := model.JobPost{}

	// Find existing job post
	if err := database.DBinstance.Where("id = ?", id).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to retrieve job post: %s", err.Error())})
		return
	}

	// Verify ownership: the job post must belong to the requesting company user
	// Compare as strings to avoid type mismatches
	if job.CompanyID.String() != user.ID.String() && user.Role != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to edit this job post"})
		return
	}

	// Bind incoming JSON to a temporary struct to avoid overwriting ownership fields
	updated := model.JobPost{}
	if err := c.ShouldBindJSON(&updated); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to parse request body: %s", err.Error())})
		return
	}

	// Preserve ID and CompanyID
	updated.ID = job.ID
	updated.CompanyID = job.CompanyID

	// Update fields on the existing job record without saving associations
	if err := database.DBinstance.Model(&job).Updates(updated).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update job post: %s", err.Error())})
		return
	}

	// Reload the job post to return the latest data
	if err := database.DBinstance.Where("id = ?", job.ID).First(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to retrieve updated job post: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, job)
}

// DeleteJobPost allows a company user to delete a job post they own.
func DeleteJobPost(c *gin.Context) {
	user := utilities.ExtractUser(c)
	id := c.Param("id")

	job := model.JobPost{}
	if err := database.DBinstance.Where("id = ?", id).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to retrieve job post: %s", err.Error())})
		return
	}

	if job.CompanyID.String() != user.ID.String() {
		// Allow admins to bypass ownership check
		if user.Role != model.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to delete this job post"})
			return
		}
	}

	if err := database.DBinstance.Delete(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to delete job post: %s", err.Error())})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job post deleted"})
}
