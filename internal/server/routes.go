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

	// Init swagger doc
	_ "HireMeMaybe-backend/docs"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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

	gAuth := auth.NewOauthLoginHandler(s.DB, googleOauth, "https://www.googleapis.com/oauth2/v2/userinfo")
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
	v1 := r.Group("/api/v1")
	{
		authRoute := v1.Group("/auth")
		{
			authRoute.POST("google/cpsk", gAuth.CPSKGoogleLoginHandler)
			authRoute.POST("google/company", gAuth.CompanyGoogleLoginHandler)
			authRoute.POST("google/visitor", gAuth.VisitorGoogleLoginHandler)
			authRoute.GET("google/callback", gAuth.Callback)

			authRoute.POST("login", lAuth.LocalLoginHandler)
			authRoute.POST("register", lAuth.LocalRegisterHandler)
		}
		// Any routes
		needAuth := v1.Group("")
		{
			needAuth.Use(middleware.RequireAuth(s.DB), middleware.CheckPunishment(s.DB, model.BanPunishment))
			file := needAuth.Group("/file")
			{
				file.GET(":id", controller.GetFile)
			}

			companyRoute := needAuth.Group("/company")
			{
				companyRoute.GET(":company_id", controller.GetCompanyByID) // New route: same handler, different path
				companyRoute.Use(middleware.CheckRole(model.RoleCompany))
				companyRoute.PATCH("profile", controller.EditCompanyProfile)
				companyRoute.POST("profile/logo", middleware.SizeLimit(10<<20), controller.UploadLogo)
				companyRoute.POST("profile/banner", middleware.SizeLimit(10<<20), controller.UploadBanner)
				companyRoute.GET("myprofile", controller.GetCompanyProfile)
				companyRoute.POST("ai-verify", controller.AIVerifyCompany)
			}

			// Job post endpoints (company only) 
			jobPostRoute := needAuth.Group("/jobpost")
			{
				jobPostRoute.GET("/:id", controller.GetPostByID)
				jobPostRoute.GET("", controller.GetPosts)
				jobPostRoute.Use(middleware.CheckRole(model.RoleCompany), middleware.CheckPunishment(s.DB, model.SuspendPunishment))
				jobPostRoute.POST("", controller.CreateJobPostHandler)

			}

			// Reporting endpoints
			reportRoute := needAuth.Group("/report")
			{
				reportRoute.PUT("/:type/:id", middleware.CheckRole(model.RoleAdmin), controller.UpdateReportStatus)
				reportRoute.GET("", middleware.CheckRole(model.RoleAdmin), controller.GetReport)
				reportRoute.POST("/user", controller.CreateUserReport)
				reportRoute.POST("/post", middleware.CheckRole(model.RoleCPSK), controller.CreatePostReport)
			}

			needCompanyAdmin := needAuth.Group("")
			{
				needCompanyAdmin.Use(middleware.CheckRole(model.RoleAdmin, model.RoleCompany))
				needCompanyAdmin.PATCH("jobpost/:id", controller.EditJobPost)
				needCompanyAdmin.DELETE("jobpost/:id", controller.DeleteJobPost)
			}

			needAdmin := needAuth.Group("")
			{
				needAdmin.Use(middleware.CheckRole(model.RoleAdmin))
				needAdmin.GET("get-companies", controller.GetCompanies)
				needAdmin.PATCH("verify-company/:company_id", controller.VerifyCompany)
				needAdmin.PUT("punish/:user_id", controller.PunishUser)
			}

			// CPSK routes: apply role check once for all CPSK endpoints
			needCPSK := needAuth.Group("")
			{
				needCPSK.Use(middleware.CheckRole(model.RoleCPSK))
				cpskRoute := needCPSK.Group("/cpsk")
				{
					cpskRoute.PATCH("profile", controller.EditCPSKProfile)
					cpskRoute.GET("myprofile", controller.GetMyCPSKProfile)
					cpskRoute.POST("profile/resume", middleware.SizeLimit(10<<20), controller.UploadResume)
				}

				needCPSK.Use(middleware.CheckPunishment(s.DB, model.SuspendPunishment))
				needCPSK.POST("application", controller.ApplicationHandler)
			}

		}
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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
