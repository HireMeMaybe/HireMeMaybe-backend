// Package server contain implementation of go-gin-server and each route handlers
package server

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/controller"
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/middleware"
	"HireMeMaybe-backend/internal/model"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	// Load env
	_ "github.com/joho/godotenv/autoload"
)

// RegisterRoutes will register each http endpoint routes to bound Server instance
func RegisterRoutes() http.Handler {
	r := gin.Default()

	allowOrginsStr := os.Getenv("ALLOW_ORIGIN")
	allowOrgins := strings.Split(allowOrginsStr, ",")

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrgins, // Add your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	r.GET("/", HelloWorldHandler)
	r.GET("/health", healthHandler)

	r.POST("/auth/google/cpsk", auth.CPSKGoogleLoginHandler)
	r.POST("/auth/google/company", auth.CompanyGoogleLoginHandler)
	r.GET("/auth/google/callback", auth.Callback)

	needAuth := r.Use(middleware.RequireAuth())

	needAuth.PUT("/cpsk/profile", middleware.CheckRole(model.RoleCPSK), controller.EditCPSKProfile)
	needAuth.GET("/cpsk/myprofile", middleware.CheckRole(model.RoleCPSK), controller.GetMyCPSKProfile)
	needAuth.POST("/cpsk/profile/resume", middleware.CheckRole(model.RoleCPSK), middleware.SizeLimit(10<<20), controller.UploadResume)

	needAuth.GET("/company/myprofile", controller.GetCompanyProfile)
	needAuth.GET("/company/profile/:company_id", controller.GetCompanyByID)
	needAuth.GET("/company/:company_id", controller.GetCompanyByID) // New route: same handler, different path
	needAuth.PUT("/company/profile", controller.EditCompanyProfile)
	needAuth.POST("/company/profile/logo", middleware.SizeLimit(10<<20), controller.UploadLogo)
	needAuth.POST("/company/profile/banner", middleware.SizeLimit(10<<20), controller.UploadBanner)

	// Job post endpoints (company only)
	needAuth.GET("/jobpost", controller.GetAllPost)
	needAuth.POST("/jobpost", middleware.CheckRole(model.RoleCompany), controller.CreateJobPostHandler)
	needAuth.PUT("/jobpost/:id", middleware.CheckRole(model.RoleCompany), controller.EditJobPost)
	needAuth.DELETE("/jobpost/:id", middleware.CheckRole(model.RoleCompany), controller.DeleteJobPost)

	needAuth.GET("/file/:id", controller.GetFile)

	return r
}

// HelloWorldHandler handle request by return message "Hello World"
func HelloWorldHandler(c *gin.Context) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	c.JSON(http.StatusOK, resp)
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, database.Health())
}