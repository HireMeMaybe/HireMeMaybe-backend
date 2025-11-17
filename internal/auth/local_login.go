package auth

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// LocalRegisterHandler holds DB reference for handler methods.
type LocalRegisterHandler struct {
	DB *database.DBinstanceStruct
}

// NewLocalAuthHandler creates a new instance of LocalRegisterHandler with the provided database connection.
func NewLocalAuthHandler(db *database.DBinstanceStruct) *LocalRegisterHandler {
	return &LocalRegisterHandler{
		DB: db,
	}
}

type registerInfo struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required,oneof=cpsk company"`
}

type loginInfo struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type adminResponse struct {
	User        model.User `json:"user"`
	AccessToken string     `json:"access_token"`
}

// LocalRegisterHandler function handles local registration by receiving username and password
// do nothing if username already exist in the database
// do nothing if password is shorter than 8 characters
// @Summary Handles local registration by receiving username and password
// @Description Username must not already exist and password must longer or equal to 8 characters long
// @Tags Auth
// @Accept json
// @Produce json
// @Param Info body registerInfo true "role can be only 'cpsk' or 'company'"
// @Success 200 {object} model.CompanyResponse "If role is company"
// @Success 200 {object} model.CPSKResponse "If role is cpsk"
// @Success 200 {object} model.VisitorResponse "If role is visitor"
// @Failure 400 {object} utilities.ErrorResponse "Info provided not met the condition"
// @Failure 500 {object} utilities.ErrorResponse "Database or password hashing error"
// @Router /auth/register [post]
func (lh *LocalRegisterHandler) LocalRegisterHandler(c *gin.Context) {
	var info registerInfo

	if err := c.ShouldBindJSON(&info); err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: "Username, password, and Role (Only 'cpsk' or 'company) must be provided",
		})
		return
	}

	var user model.User
	err := lh.DB.Where("username = ?", info.Username).First(&user).Error

	switch {
	case err == nil:
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: "Username already exist",
		})
		return

	case errors.Is(err, gorm.ErrRecordNotFound):
		// Do nothing

	default:
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}

	if len(info.Password) < 8 {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: "Password should longer or equal to 8 characters",
		})
		return
	}

	hashedPassword, err := utilities.HashPassword(info.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Failed hash password: %s", err.Error()),
		})
		return
	}

	switch info.Role {
	case "cpsk":
		cpskUser := model.CPSKUser{
			User: model.User{
				Username: info.Username,
				Password: hashedPassword,
				Role:     model.RoleCPSK,
			},
		}
		if err := lh.DB.Create(&cpskUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to create user: %s", err.Error()),
			})
			return
		}

		accessToken, _, err := GenerateStandardToken(cpskUser.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to generate access token: %s", err.Error()),
			})
			return
		}

		c.JSON(http.StatusCreated, model.CPSKResponse{
			User:        cpskUser,
			AccessToken: accessToken,
		})
	case "company":
		verified := model.StatusPending
		if strings.ToLower(strings.TrimSpace(os.Getenv("BYPASS_VERIFICATION"))) == "true" {
			verified = model.StatusVerified
		}

		companyUser := model.CompanyUser{
			User: model.User{
				Username: info.Username,
				Password: hashedPassword,
				Role:     model.RoleCompany,
			},
			VerifiedStatus: verified,
		}
		if err := lh.DB.Create(&companyUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to create user: %s", err.Error()),
			})
			return
		}

		accessToken, _, err := GenerateStandardToken(companyUser.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to generate access token: %s", err.Error()),
			})
			return
		}

		c.JSON(http.StatusCreated, model.CompanyResponse{
			User:        companyUser,
			AccessToken: accessToken,
		})
	default:
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: fmt.Sprintf("Role '%s' not allowed", info.Role),
		})
	}
}

// LocalLoginHandler function handles local login by receiving username and password
// do nothing if username does not exist in the database
// do nothing if password is incorrect
// @Summary Handles local login by receiving username and password
// @Description Username must exist and password match
// @Tags Auth
// @Accept json
// @Produce json
// @Param Info body loginInfo true "Credentials for login"
// @Success 200 {object} model.CompanyResponse "If role is company"
// @Success 200 {object} model.CPSKResponse "If role is cpsk"
// @Failure 400 {object} utilities.ErrorResponse "Info provided not met the condition"
// @Failure 401 {object} utilities.ErrorResponse "Username not exist or password incorrect"
// @Failure 500 {object} utilities.ErrorResponse "Database or password hashing error"
// @Router /auth/login [post]
func (lh *LocalRegisterHandler) LocalLoginHandler(c *gin.Context) {
	var info loginInfo

	if err := c.ShouldBindJSON(&info); err != nil {
		c.JSON(http.StatusBadRequest, utilities.ErrorResponse{
			Error: "Username or password is not provided",
		})
		return
	}

	var user model.User
	err := lh.DB.Preload("Punishment").Where("username = ?", info.Username).First(&user).Error

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusUnauthorized, utilities.ErrorResponse{
			Error: "Username or password is incorrect",
		})
		return

	case err == nil:
		// Do nothing

	default:
		c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
			Error: fmt.Sprintf("Database error: %s", err.Error()),
		})
		return
	}

	if msg, status, err := database.RemovePunishment(user, lh.DB); err != nil {
		if status == http.StatusInternalServerError {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: msg,
			})
			return
		}
	}

	if user.Password == "" {
		c.JSON(http.StatusUnauthorized, utilities.ErrorResponse{
			Error: "Username or password is incorrect",
		})
		return
	}

	if !utilities.VerifyPassword(info.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, utilities.ErrorResponse{
			Error: "Username or password is incorrect",
		})
		return
	}

	switch user.Role {
	case model.RoleCPSK:
		var cpskUser model.CPSKUser
		if err := lh.DB.Preload("User").Preload("User.Punishment").Where("user_id = ?", user.ID).First(&cpskUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to retrieve user data: %s", err.Error()),
			})
			return
		}

		accessToken, _, err := GenerateStandardToken(cpskUser.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to generate access token: %s", err.Error()),
			})
			return
		}

		c.JSON(http.StatusOK, model.CPSKResponse{
			User:        cpskUser,
			AccessToken: accessToken,
		})
	case model.RoleCompany:
		var companyUser model.CompanyUser
		if err := lh.DB.Preload("User").Preload("User.Punishment").Where("user_id = ?", user.ID).First(&companyUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to retrieve user data: %s", err.Error()),
			})
			return
		}

		accessToken, _, err := GenerateStandardToken(companyUser.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to generate access token: %s", err.Error()),
			})
			return
		}

		c.JSON(http.StatusOK, model.CompanyResponse{
			User:        companyUser,
			AccessToken: accessToken,
		})
	default:
		accessToken, _, err := GenerateStandardToken(user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, utilities.ErrorResponse{
				Error: fmt.Sprintf("Failed to generate access token: %s", err.Error()),
			})
			return
		}

		c.JSON(http.StatusOK, adminResponse{
			User:        user,
			AccessToken: accessToken,
		})
	}
}
