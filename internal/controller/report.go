package controller

import (
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type reportRequest struct {
	ReportedID string `json:"reported_id" binding:"required,uuid"`
	Reason     string `json:"reason" binding:"required"`
}

// CreateUserReport handles the creation of a report against a user.
// @Summary Create a report against a user
// @Description Create a report against a user. Cannot report admins or users with the same role as the reporter.
// @Tags Report
// @Accept json
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param report body reportRequest true "Report information"
// @Success 201 {object} utilities.MessageResponse "Report created successfully"
// @Failure 400 {object} utilities.ErrorResponse "Invalid request body, reported user not found, cannot report this user"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "User doesn't have permission to access"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /report/user [post]
func (jc *JobController) CreateUserReport(c *gin.Context) {
	user := utilities.ExtractUser(c)
	if c.IsAborted() {
		return
	}

	var req reportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, utilities.ErrorResponse{
			Error: "Invalid request body",
		})
		return
	}
	
	reportedUUID, err := uuid.Parse(req.ReportedID)
	if err != nil {
		c.JSON(400, utilities.ErrorResponse{
			Error: "Invalid reported_id format",
		})
		return
	}

	reportedUser := model.User{}
	if err := jc.DB.Where("id = ?", reportedUUID).First(&reportedUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(400, utilities.ErrorResponse{
				Error: "Reported user not found",
			})
			return
		}
		c.JSON(500, utilities.ErrorResponse{
			Error: "Database error: " + err.Error(),
		})
		return
	}

	if reportedUser.Role == model.RoleAdmin || user.Role == reportedUser.Role {
		c.JSON(400, utilities.ErrorResponse{
			Error: "You cannot report this user",
		})
		return
	}

	report := model.ReportOnUser{
		ReportedUserID: reportedUUID,
		ReportCommon: model.ReportCommon{
			Reporter: user.ID,
			Reason:     req.Reason,
		},
	}

	if err := jc.DB.Create(&report).Error; err != nil {
		c.JSON(500, utilities.ErrorResponse{
			Error: "Database error: " + err.Error(),
		})
		return
	}

	c.JSON(201, gin.H{
		"message": "Report created successfully",
	})
}

// CreatePostReport handles the creation of a report against a job post.
// @Summary Create a report against a job post
// @Description Create a report against a job post.
// @Tags Report
// @Accept json
// @Produce json
// @Param Authorization header string true "Insert your access token" default(Bearer <your access token>)
// @Param report body reportRequest true "Report information"
// @Success 201 {object} utilities.MessageResponse "Report created successfully"
// @Failure 400 {object} utilities.ErrorResponse "Invalid request body, reported post not found"
// @Failure 401 {object} utilities.ErrorResponse "Invalid token"
// @Failure 403 {object} utilities.ErrorResponse "User doesn't have permission to access"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /report/post [post]
func (jc *JobController) CreatePostReport(c *gin.Context) {

	user := utilities.ExtractUser(c)
	if c.IsAborted() {
		return
	}

	var req reportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, utilities.ErrorResponse{
			Error: "Invalid request body",
		})
		return
	}

	postID, err := strconv.ParseUint(req.ReportedID, 10, 64)
	if err != nil {
		c.JSON(400, utilities.ErrorResponse{
			Error: "Invalid reported_id format",
		})
		return
	}

	dberr := jc.DB.Where("id = ?", postID).First(&model.JobPost{}).Error
	if dberr != nil {
		if errors.Is(dberr, gorm.ErrRecordNotFound) {
			c.JSON(400, utilities.ErrorResponse{
				Error: "Reported post not found",
			})
			return
		}
		c.JSON(500, utilities.ErrorResponse{
			Error: "Database error: " + dberr.Error(),
		})
		return
	}

	report := model.ReportOnPost{
		ReportedPostID: uint(postID),
		ReportCommon: model.ReportCommon{
			Reporter: user.ID,
			Reason:   req.Reason,
		},
	}

	if err := jc.DB.Create(&report).Error; err != nil {
		c.JSON(500, utilities.ErrorResponse{
			Error: "Database error: " + err.Error(),
		})
		return
	}

	c.JSON(201, gin.H{
		"message": "Report created successfully",
	})
}