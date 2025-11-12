// Package jobpost provides HTTP handlers for job post related operations.
package jobpost

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// JobPostController handles job post related endpoints
type JobPostController struct {
	DB *database.DBinstanceStruct
}

// NewJobPostController creates a new instance of JobPostController
func NewJobPostController(db *database.DBinstanceStruct) *JobPostController {
	return &JobPostController{
		DB: db,
	}
}

// CreateJobPostHandler handles the creation of a new job post by a company user.
// @Summary Create job post based on given json structure
// @Description Only verified company have access to this endpoint
// @Tags Jobpost
// @Accept json
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param Jobpost body model.EditableJobPostInfo true "Input jobpost information"
// @Success 201 {object} model.JobPost "Successfully create job post"
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header, or invalid job post struct"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Not logged in as verified company, User is banned or suspended"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /jobpost [post]
func (jc *JobPostController) CreateJobPostHandler(c *gin.Context) {

	// Get user
	user := utilities.ExtractUser(c)

	// Ensure that user is a verified company
	var companyUser model.CompanyUser
	if err := jc.DB.Where("user_id = ?", user.ID.String()).First(&companyUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusForbidden, utilities.ErrorResponse{Error: "Only company users can create job posts"})
			return
		}
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to retrieve company information: %s", err.Error()),
		})
		return
	}
	if companyUser.VerifiedStatus != model.StatusVerified {
		c.JSON(http.StatusForbidden, utilities.ErrorResponse{
			Error: "Only verified companies can create job posts",
		})
		return
	}

	// construct job post from request
	jobPost := model.JobPost{}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&jobPost.EditableJobPostInfo); err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: fmt.Sprintf("Invalid request body: %s", err.Error()),
		})
		return
	}

	// save job post
	jobPost.CompanyUserID = user.ID
	if err := jc.DB.Create(&jobPost).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprint("Failed to create job post: ", err),
		})
		return
	}

	// response
	c.JSON(http.StatusCreated, jobPost)
}

// GetPosts fetches all non-expired job posts that match query from the database
// and returns them as a JSON response.
// @Summary Get non-expired job posts based on query
// @Description Every query are not required, but they have specific use defined in their description
// @Tags Jobpost
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param search query string false "Search from job post title with substring matching and case insensitive"
// @Param type query string false "Job type field with substring matching and case insensitive"
// @Param tag query string false "Search if tags field contain tag param, no substring matching and case insensitive"
// @Param salary query string false "Salary field, must exactly match to get result"
// @Param exp query string false "Exp_lvl field, must exactly match to get result"
// @Param company query string false "Search from company name with substring matching and case insensitive"
// @Param industry query string false "Search from industry of company with substring matching and case insensitive"
// @Param location query string false "Search from location with substring matching and case insensitive"
// @Param desc query boolean false "Sorting by post time in descending if true, otherwise ascendind"
// @Success 200 {array} model.JobPostResponse "Return non-expired job post(s)"
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header, or invalid job post struct"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "User is banned"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /jobpost [get]
func (jc *JobPostController) GetPosts(c *gin.Context) {

	user := utilities.ExtractUser(c)
	if c.IsAborted() {
		return
	}

	rawSearch := c.Query("search")
	rawJobType := c.Query("type")
	rawTag := c.Query("tag")
	rawSalary := c.Query("salary")
	rawExp := c.Query("exp")
	rawCompany := c.Query("company")
	rawIndustry := c.Query("industry")
	rawLocation := c.Query("location")
	rawDesc := c.Query("desc")

	var rawPosts []model.JobPost

	result := jc.DB.Preload("CompanyUser").
		Preload("CompanyUser.User").
		Preload("CompanyUser.User.Punishment").
		Preload("Applications").
		Where("expiring > ? OR expiring IS NULL", time.Now())

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

	if rawExp != "" {
		result = result.Where("exp_lvl = ?", rawExp)
	}

	// Join company_users table only once if needed for company or industry filters
	if rawCompany != "" || rawIndustry != "" {
		result = result.Joins("JOIN company_users ON company_users.user_id = job_posts.company_user_id")
	}

	if rawCompany != "" {
		result = result.Where("company_users.name ILIKE ?", "%"+rawCompany+"%")
	}

	if rawIndustry != "" {
		result = result.Where("company_users.industry ILIKE ?", "%"+rawIndustry+"%")
	}

	if rawLocation != "" {
		result = result.Where("job_posts.location ILIKE ?", "%"+rawLocation+"%")
	}

	result = result.Order(clause.OrderByColumn{
			Column: clause.Column{Name: "post_time"},
			Desc:   strings.ToLower(rawDesc) == "true",
		}).Find(&rawPosts)
		
	if err := result.Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprint("Failed to fetch job post: ", err.Error()),
		})
		return
	}

	posts := []model.JobPostResponse{}
	for _, rawPost := range rawPosts {
		if rawPost.CompanyUser.User.Punishment != nil {
			if rawPost.CompanyUser.User.Punishment.PunishmentType == "ban" &&
				(rawPost.CompanyUser.User.Punishment.PunishEnd == nil ||
					rawPost.CompanyUser.User.Punishment.PunishEnd.After(time.Now())) {
				continue
			}
		}
		rawPostResp, err := rawPost.ToJobPostResponse(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprint("Failed to process job post: ", err.Error()),
			})
			return
		}
		posts = append(posts, rawPostResp)
	}


	c.JSON(http.StatusOK, posts)
}

// GetPostByID fetches a job post by its ID from the database
// and returns it as a JSON response.
// @Summary Get job post by ID
// @Description Retrieve a specific job post using its unique ID
// @Tags Jobpost
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param id path integer true "ID of desired job post"
// @Success 200 {object} model.JobPostResponse "Return the job post with the specified ID"
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "User is banned"
// @Failure 404 {object} utilities.ErrorResponse "Job post not found"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /jobpost/{id} [get]
func (jc *JobPostController) GetPostByID(c *gin.Context) {
	id := c.Param("id")

	user := utilities.ExtractUser(c)
	if c.IsAborted() {
		return
	}

	job := model.JobPost{}
	if err := jc.DB.
		Preload("CompanyUser").
		Preload("CompanyUser.User").
		Preload("Applications").
		Where("id = ?", id).
		First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, utilities.ErrorResponse{Error: "Job post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to retrieve job post: %s", err.Error()),
		})
		return
	}

	rawPostResp, err := job.ToJobPostResponse(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprint("Failed to process job post: ", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, rawPostResp)
}

// EditJobPost allows a company user to update a job post they own.
// @Summary Edit job post based on given json structure
// @Description Only company that own the post or admin have access to this endpoint
// @Tags Jobpost
// @Accept json
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param id path integer true "ID of desired job post"
// @Param Jobpost body model.EditableJobPostInfo true "Input jobpost information"
// @Success 200 {object} model.JobPost "Successfully update job post"
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header, or invalid job post struct"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Do not have permission to edit, User is banned"
// @Failure 404 {object} utilities.ErrorResponse "Post not found"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /jobpost/{id} [patch]
func (jc *JobPostController) EditJobPost(c *gin.Context) {

	// Use ExtractUser itiesity to get authenticated user
	user := utilities.ExtractUser(c)

	// Get job post id from path
	id := c.Param("id")

	job := model.JobPost{}

	// Find existing job post
	if err := jc.DB.Where("id = ?", id).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, utilities.ErrorResponse{Error: "Job post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to retrieve job post: %s", err.Error()),
		})
		return
	}

	// Verify ownership: the job post must belong to the requesting company user
	// Compare as strings to avoid type mismatches
	if job.CompanyUserID.String() != user.ID.String() {
		c.JSON(http.StatusForbidden, utilities.ErrorResponse{
			Error: "You are not allowed to edit this job post",
		})
		return
	}

	// Bind incoming JSON to a temporary struct to avoid overwriting ownership fields
	updated := model.JobPost{}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&updated.EditableJobPostInfo); err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to parse request body: %s", err.Error()),
		})
		return
	}

	// Update fields on the existing job record without saving associations
	if err := jc.DB.Model(&job).Updates(updated).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to update job post: %s", err.Error()),
		})
		return
	}

	// Reload the job post to return the latest data
	if err := jc.DB.Where("id = ?", job.ID).First(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to retrieve updated job post: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, job)
}

// DeleteJobPost allows a company user to delete a job post they own.
// @Summary Delete given job post ID
// @Description Only company that own the post or admin have access to this endpoint
// @Tags Jobpost
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param id path integer true "ID of desired job post"
// @Success 200 {object} utilities.MessageResponse "Successfully delete job post"
// @Failure 400 {object} utilities.ErrorResponse "Invalid authorization header, or invalid job post struct"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "Do not have permission to delete this post, User is banned"
// @Failure 404 {object} utilities.ErrorResponse "Post not found"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /jobpost/{id} [delete]
func (jc *JobPostController) DeleteJobPost(c *gin.Context) {
	user := utilities.ExtractUser(c)
	id := c.Param("id")

	job := model.JobPost{}
	if err := jc.DB.Where("id = ?", id).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, utilities.ErrorResponse{Error: "Job post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to retrieve job post: %s", err.Error()),
		})
		return
	}

	if job.CompanyUserID.String() != user.ID.String() {
		// Allow admins to bypass ownership check
		if user.Role != model.RoleAdmin {
			c.JSON(http.StatusForbidden, utilities.ErrorResponse{
				Error: "You are not allowed to delete this job post",
			})
			return
		}
	}

	if err := jc.DB.Delete(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed to delete job post: %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, utilities.MessageResponse{Message: "Job post deleted"})
}
