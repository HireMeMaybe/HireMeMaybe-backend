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

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title HireMeMaybe API service
// @version 1.0
// @description This is HireMeMaybe API service that provide data for HireMeMaybe web app
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
	v1 := r.Group("/api/v1")
	{
		authRoute := v1.Group("/auth")
		{
			authRoute.POST("google/cpsk", auth.CPSKGoogleLoginHandler)
			authRoute.POST("google/company", auth.CompanyGoogleLoginHandler)
			authRoute.GET("google/callback", auth.Callback)

			authRoute.POST("login", auth.LocalLoginHandler)
			authRoute.POST("register", auth.LocalRegisterHandler)
		}
		// Any routes
		needAuth := v1.Group("")
		{
			needAuth.Use(middleware.RequireAuth())
			file := needAuth.Group("/file")
			{
				file.GET(":id", controller.GetFile)
			}

			companyRoute := needAuth.Group("/company")
			{
				companyRoute.GET("profile/:company_id", controller.GetCompanyByID)
				companyRoute.GET(":company_id", controller.GetCompanyByID) // New route: same handler, different path
				companyRoute.PUT("profile", controller.EditCompanyProfile)
				companyRoute.POST("profile/logo", middleware.SizeLimit(10<<20), controller.UploadLogo)
				companyRoute.POST("profile/banner", middleware.SizeLimit(10<<20), controller.UploadBanner)
				companyRoute.GET("myprofile", controller.GetCompanyProfile)
			}

			// Job post endpoints (company only)
			jobPostRoute := needAuth.Group("/jobpost")
			{
				jobPostRoute.GET("", controller.GetPosts)
				jobPostRoute.Use(middleware.CheckRole(model.RoleCompany))
				jobPostRoute.POST("", controller.CreateJobPostHandler)
				jobPostRoute.PUT(":id", controller.EditJobPost)

			}

			needCompanyAdmin := needAuth.Group("")
			{
				needCompanyAdmin.Use(middleware.CheckRole(model.RoleAdmin, model.RoleCompany))
				needCompanyAdmin.DELETE("jobpost/:id", controller.DeleteJobPost)
			}

			needAdmin := needAuth.Group("")
			{
				needAdmin.Use(middleware.CheckRole(model.RoleAdmin))
				needAdmin.GET("get-companies", controller.GetCompanies)
				needAdmin.PUT("verify-company", controller.VerifyCompany)
			}

			// CPSK routes: apply role check once for all CPSK endpoints
			needCPSK := needAuth.Group("")
			{
				needCPSK.Use(middleware.CheckRole(model.RoleCPSK))
				cpskRoute := needCPSK.Group("/cpsk")
				{
					cpskRoute.PUT("profile", controller.EditCPSKProfile)
					cpskRoute.GET("myprofile", controller.GetMyCPSKProfile)
					cpskRoute.POST("profile/resume", middleware.SizeLimit(10<<20), controller.UploadResume)
				}

				needCPSK.POST("application", controller.ApplicationHandler)
			}
		}
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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
