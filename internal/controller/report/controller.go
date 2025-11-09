// Package report provides HTTP handlers for report-related operations.
package report

import (
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"errors"
	"net/http"

	"HireMeMaybe-backend/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ReportController handles report related endpoints
type ReportController struct {
	DB *database.DBinstanceStruct
}

// NewReportController creates a new instance of ReportController
func NewReportController(db *database.DBinstanceStruct) *ReportController {
	return &ReportController{
		DB: db,
	}
}

// UserReportRequest represents the request body for reporting a user.
type UserReportRequest struct {
	ReportedID string `json:"reported_id" binding:"required,uuid"`
	Reason     string `json:"reason" binding:"required"`
}

// PostReportRequest represents the request body for reporting a job post.
type PostReportRequest struct {
	ReportedID uint   `json:"reported_id" binding:"required"`
	Reason     string `json:"reason" binding:"required"`
}

// CreateUserReport handles the creation of a report against a user.
// @Summary Create a report against a user
// @Description Create a report against a user. Cannot report admins or users with the same role as the reporter.
// @Tags Report
// @Accept json
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param report body UserReportRequest true "Report information"
// @Success 201 {object} object{message=string,report_id=integer}"Report created successfully"
// @Failure 400 {object} utilities.ErrorResponse "Invalid request body, reported user not found, cannot report this user"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "User doesn't have permission to access"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /report/user [post]
func (jc *ReportController) CreateUserReport(c *gin.Context) {
	user := utilities.ExtractUser(c)
	if c.IsAborted() {
		return
	}

	var req UserReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: "Invalid request body" + err.Error(),
		})
		return
	}

	reportedUUID, err := uuid.Parse(req.ReportedID)
	if err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: "Invalid reported_id format",
		})
		return
	}

	reportedUser := model.User{}
	if err := jc.DB.Where("id = ?", reportedUUID).First(&reportedUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, utilities.ErrorResponse{
				Error: "Reported user not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: "Database error: " + err.Error(),
		})
		return
	}

	if reportedUser.Role == model.RoleAdmin || user.Role == reportedUser.Role {
		c.JSON(http.StatusForbidden, utilities.ErrorResponse{
			Error: "You cannot report this user",
		})
		return
	}

	report := model.ReportOnUser{
		ReportedUserID: reportedUUID,
		ReportCommon: model.ReportCommon{
			Reporter: user.ID,
			Reason:   req.Reason,
		},
	}

	if err := jc.DB.Create(&report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: "Database error: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Report created successfully",
		"report_id": report.ID,
	})
}

// CreatePostReport handles the creation of a report against a job post.
// @Summary Create a report against a job post
// @Description Create a report against a job post.
// @Tags Report
// @Accept json
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param report body PostReportRequest true "Report information"
// @Success 201 {object} object{message=string,report_id=integer}"Report created successfully"
// @Failure 400 {object} utilities.ErrorResponse "Invalid request body, reported post not found"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "User doesn't have permission to access"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /report/post [post]
func (jc *ReportController) CreatePostReport(c *gin.Context) {

	user := utilities.ExtractUser(c)
	if c.IsAborted() {
		return
	}

	var req PostReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: "Invalid request body" + err.Error(),
		})
		return
	}

	dberr := jc.DB.Where("id = ?", req.ReportedID).First(&model.JobPost{}).Error
	if dberr != nil {
		if errors.Is(dberr, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, utilities.ErrorResponse{
				Error: "Reported post not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: "Database error: " + dberr.Error(),
		})
		return
	}

	report := model.ReportOnPost{
		ReportedPostID: uint(req.ReportedID),
		ReportCommon: model.ReportCommon{
			Reporter: user.ID,
			Reason:   req.Reason,
		},
	}

	if err := jc.DB.Create(&report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: "Database error: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message":   "Report created successfully",
		"report_id": report.ID,
	})
}

// GetReport retrieves reports from the database, optionally filtered by status.
// @Summary Retrieve reports from database
// @Description Retrieve reports from database, optionally filtered by status.
// @Tags Report
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param status query string false "Filter reports by status (e.g., pending, reviewed, resolved)"
// @Success 200 {object} map[string]interface{} "Successfully retrieve reports"
// @Failure 400 {object} utilities.ErrorResponse "Invalid request parameters"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "User doesn't have permission to access"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /report [get]
func (jc *ReportController) GetReport(c *gin.Context) {
	reportStatus := c.Query("status")

	var postReports []model.ReportOnPost
	var userReports []model.ReportOnUser

	// If status is provided, filter by status
	postQuery := jc.DB.Preload("ReporterUser").Preload("ReportedPost")
	userQuery := jc.DB.Preload("ReporterUser").Preload("ReportedUser")

	if reportStatus != "" {
		postQuery = postQuery.Where("status = ?", reportStatus)
		userQuery = userQuery.Where("status = ?", reportStatus)
	}

	if err := postQuery.Find(&postReports).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: "Database error: " + err.Error(),
			})
			return
		}
	}

	if err := userQuery.Find(&userReports).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: "Database error: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"post_reports": postReports,
		"user_reports": userReports,
	})
}

// UpdateReportStatus updates the status of a report (either on a user or a post).
// @Summary Update the status of a report
// @Description Update the status of a report (either on a user or a post).
// @Tags Report
// @Accept json
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param id path string true "ID of the report to update"
// @Param type path string true "Type of report to update (user or post)"
// @Param report body object{status=string,admin_note=string} true "Report status update information"
// @Success 200 {object} utilities.MessageResponse "Report status updated successfully"
// @Failure 400 {object} utilities.ErrorResponse "Invalid request body, report not found"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "User doesn't have permission to access"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /report/{type}/{id} [put]
func (jc *ReportController) UpdateReportStatus(c *gin.Context) {
	reportID := c.Param("id")
	rType := c.Param("type")

	var req struct {
		Status    string `json:"status" binding:"required,oneof=pending resolved rejected"`
		AdminNote string `json:"admin_note"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: "Invalid request body",
		})
		return
	}

	var report model.UpdateableReport

	switch rType {
	case "user":
		var userReport model.ReportOnUser
		if err := jc.DB.Where("id = ?", reportID).First(&userReport).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
					Error: "Report not found",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: "Database error: " + err.Error(),
			})
			return
		}
		report = &userReport
	case "post":
		var postReport model.ReportOnPost
		if err := jc.DB.Where("id = ?", reportID).First(&postReport).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
					Error: "Report not found",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: "Database error: " + err.Error(),
			})
			return
		}
		report = &postReport
	default:
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: "Invalid report type",
		})
		return
	}

	err := report.UpdateStatus(req.Status, req.AdminNote)
	if err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: "Invalid status value",
		})
		return
	}
	if err := jc.DB.Save(report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: "Database error: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, utilities.MessageResponse{
		Message: "Report status updated successfully",
	})
}
