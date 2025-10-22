package auth

import (
	"HireMeMaybe-backend/internal/model"
	"net/http"

	"github.com/gin-gonic/gin"
	// Auto load .env file
	_ "github.com/joho/godotenv/autoload"
)

// CPSKGoogleLoginHandler handles Google login authentication for cpsk role, exchanges code for user
// info, checks and creates user in the database, generates an access token, and returns user
// information with the access token.
// @Summary Handles Google login authentication for cpsk role, exchanges code for user
// @Description Checks and creates user in the database, generates an access token
// @Tags Auth
// @Accept json
// @Produce json
// @Param Code body code true "Authentication code from google"
// @Success 200 {object} model.CPSKResponse "Login success"
// @Success 201 {object} model.CPSKResponse "Register success"
// @Failure 400 {object} utilities.ErrorResponse "Fail to receive token or fetch user info"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /auth/google/cpsk [post]
func (h *OauthLoginHandler) CPSKGoogleLoginHandler(c *gin.Context) {

	uInfo, err := h.getUserInfo(c)
	if err != nil {
		return
	}

	h.loginOrRegisterUser(&model.CPSKUser{}, uInfo, c)
}

// CompanyGoogleLoginHandler handles Google login authentication for company role, exchanges code for user
// info, checks and creates user in the database, generates an access token, and returns user
// information with the access token.
// @Summary Handles Google login authentication for company role, exchanges code for user
// @Description Checks and creates user in the database, generates an access token
// @Tags Auth
// @Accept json
// @Produce json
// @Param Code body code true "Authentication code from google"
// @Success 200 {object} model.CompanyResponse "Login success"
// @Success 201 {object} model.CompanyResponse "Register success"
// @Failure 400 {object} utilities.ErrorResponse "Fail to receive token or fetch user info"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /auth/google/company [post]
func (h *OauthLoginHandler) CompanyGoogleLoginHandler(c *gin.Context) {

	uInfo, err := h.getUserInfo(c)
	if err != nil {
		return
	}

	h.loginOrRegisterUser(&model.CompanyUser{}, uInfo, c)
}

// VisitorGoogleLoginHandler handles Google login authentication for visitor role, exchanges code for user
// info, checks and creates user in the database, generates an access token, and returns user
// information with the access token.
// @Summary Handles Google login authentication for visitor role, exchanges code for user
// @Description Checks and creates user in the database, generates an access token
// @Tags Auth
// @Accept json
// @Produce json
// @Param Code body code true "Authentication code from google"
// @Success 200 {object} model.VisitorResponse "Login success"
// @Success 201 {object} model.VisitorResponse "Register success"
// @Failure 400 {object} utilities.ErrorResponse "Fail to receive token or fetch user info"
// @Failure 500 {object} utilities.ErrorResponse "Database error"
// @Router /auth/google/visitor [post]
func (h *OauthLoginHandler) VisitorGoogleLoginHandler(c *gin.Context) {

	uInfo, err := h.getUserInfo(c)
	if err != nil {
		return
	}

	h.loginOrRegisterUser(&model.VisitorUser{}, uInfo, c)
}

// Callback function in Go retrieves a query parameter named "code" from the request and returns it
// in a JSON response.
// @Summary Retrieves a query parameter named "code" from the request and returns it in a JSON response
// @Tags Auth
// @Produce json
// @Param Code query string false "Authentication code from google"
// @Success 200 {object} code
// @Router /auth/google/callback [get]
func (h *OauthLoginHandler) Callback(c *gin.Context) {
	aCode := c.Query("code")
	c.JSON(http.StatusOK, code{
		Code: aCode,
	})
}
