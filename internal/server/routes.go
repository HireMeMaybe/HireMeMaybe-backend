// Package server contain implementation of go-gin-server and each route handlers
package server

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/controller"
	"HireMeMaybe-backend/internal/middleware"
	"HireMeMaybe-backend/internal/model"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	// Load env
	_ "github.com/joho/godotenv/autoload"
)

// RegisterRoutes will register each http endpoint routes to bound Server instance
func (s *MyServer) RegisterRoutes() http.Handler {
	r := gin.Default()

	allowOrginsStr := os.Getenv("ALLOW_ORIGIN")
	allowOrgins := strings.Split(allowOrginsStr, ",")

	googleOauth := &oauth2.Config{
		ClientID:     os.Getenv("CPSK_GOOGLE_AUTH_CLIENT"),
		ClientSecret: os.Getenv("CPSK_GOOGLE_AUTH_SECRET"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.openid",
		},
		Endpoint:    google.Endpoint,
		RedirectURL: os.Getenv("OAUTH_REDIRECT_URL"),
	}

	gAuth := auth.NewOauthLoginHandler(s.DB, googleOauth)
	lAuth := auth.NewLocalAuthHandler(s.DB)
	controller := controller.NewJobController(s.DB)

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrgins, // Add your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	r.GET("/", s.HelloWorldHandler)
	r.GET("/health", s.healthHandler)

	r.POST("/auth/google/cpsk", gAuth.CPSKGoogleLoginHandler)
	r.POST("/auth/google/company", gAuth.CompanyGoogleLoginHandler)
	r.GET("/auth/google/callback", gAuth.Callback)

	r.POST("/auth/login", lAuth.LocalLoginHandler)
	r.POST("/auth/register", lAuth.LocalRegisterHandler)

	needAuth := r.Use(middleware.RequireAuth(s.DB))

	needAuth.GET("/company/profile/:company_id", controller.GetCompanyByID)
	needAuth.GET("/company/:company_id", controller.GetCompanyByID) // New route: same handler, different path
	needAuth.PUT("/company/profile", controller.EditCompanyProfile)
	needAuth.POST("/company/profile/logo", middleware.SizeLimit(10<<20), controller.UploadLogo)
	needAuth.POST("/company/profile/banner", middleware.SizeLimit(10<<20), controller.UploadBanner)

	// Job post endpoints (company only)
	needAuth.GET("/company/myprofile", controller.GetCompanyProfile)

	needAuth.GET("/jobpost", controller.GetPosts)
	needAuth.POST("/jobpost", middleware.CheckRole(model.RoleCompany), controller.CreateJobPostHandler)

	needAuth.PUT("/jobpost/:id", middleware.CheckRole(model.RoleCompany), controller.EditJobPost)
	needAuth.DELETE("/jobpost/:id", middleware.CheckRole(model.RoleCompany, model.RoleAdmin), controller.DeleteJobPost)

	needAuth.GET("/file/:id", controller.GetFile)

	needAuth.GET("/get-companies", middleware.CheckRole(model.RoleAdmin), controller.GetCompanies)
	needAuth.PUT("/verify-company", middleware.CheckRole(model.RoleAdmin), controller.VerifyCompany)

	// CPSK routes: apply role check once for all CPSK endpoints
	needCPSK := needAuth.Use(middleware.CheckRole(model.RoleCPSK))

	needCPSK.PUT("/cpsk/profile", controller.EditCPSKProfile)
	needCPSK.GET("/cpsk/myprofile", controller.GetMyCPSKProfile)
	needCPSK.POST("/cpsk/profile/resume", middleware.SizeLimit(10<<20), controller.UploadResume)
	needCPSK.POST("/application", controller.ApplicationHandler)
	return r
}

// HelloWorldHandler handle request by return message "Hello World"
func (s *MyServer) HelloWorldHandler(c *gin.Context) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	c.JSON(http.StatusOK, resp)
}

func (s *MyServer) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.DB.Health())
}
