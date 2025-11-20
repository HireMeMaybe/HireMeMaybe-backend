// Package server contain implementation of go-gin-server and each route handlers
package server

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/controller/admin"
	"HireMeMaybe-backend/internal/controller/application"
	"HireMeMaybe-backend/internal/controller/company"
	"HireMeMaybe-backend/internal/controller/cpsk"
	"HireMeMaybe-backend/internal/controller/file"
	"HireMeMaybe-backend/internal/controller/jobpost"
	"HireMeMaybe-backend/internal/controller/punishment"
	"HireMeMaybe-backend/internal/controller/report"
	"HireMeMaybe-backend/internal/controller/verification"

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

	cloudStorageClient, err := file.NewCloudStorageClient(os.Getenv("CLOUD_STORAGE_BUCKET"))

	if err != nil {
		panic("Failed to create cloud storage client: " + err.Error())
	}

	gAuth := auth.NewOauthLoginHandler(s.DB, googleOauth, "https://www.googleapis.com/oauth2/v3/userinfo")
	lAuth := auth.NewLocalAuthHandler(s.DB)
	// controller := controller.NewJobController(s.DB)

	fileController := file.NewFileController(s.DB, cloudStorageClient)
	companyController := company.NewCompanyController(s.DB)
	adminController := admin.NewAdminController(s.DB)
	applicationController := application.NewApplicationController(s.DB)
	cpskController := cpsk.NewCPSKController(s.DB)
	jobPostController := jobpost.NewJobPostController(s.DB)
	punishmentController := punishment.NewPunishmentController(s.DB)
	reportController := report.NewReportController(s.DB)
	verificationController := verification.NewVerificationController(s.DB)

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrgins, // Add your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Enable cookies/auth
	}))


	r.GET("/", s.HelloWorldHandler)
	r.GET("/health", s.healthHandler)
	v1 := r.Group("/api/v1")
	// Apply rate limiting only to API routes (exclude Swagger and other non-API endpoints)
	v1.Use(middleware.EnvRateLimitMiddleware())
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
			fileRoute := needAuth.Group("/file")
			{
				fileRoute.GET(":id", fileController.GetFile)
			}

			companyRoute := needAuth.Group("/company")
			{
				companyRoute.GET(":company_id", companyController.GetCompanyByID) // New route: same handler, different path
				companyRoute.Use(middleware.CheckRole(model.RoleCompany))
				companyRoute.PATCH("profile", companyController.EditCompanyProfile)
				companyRoute.POST("profile/logo", middleware.SizeLimit(10<<20), fileController.UploadLogo)
				companyRoute.POST("profile/banner", middleware.SizeLimit(10<<20), fileController.UploadBanner)
				companyRoute.GET("myprofile", companyController.GetMyCompanyProfile)
				companyRoute.POST("ai-verify", verificationController.AIVerifyCompany)
			}

			// Job post endpoints (company only)
			jobPostRoute := needAuth.Group("/jobpost")
			{
				jobPostRoute.GET("/:id", jobPostController.GetPostByID)
				jobPostRoute.GET("", jobPostController.GetPosts)
				jobPostRoute.Use(middleware.CheckRole(model.RoleCompany), middleware.CheckPunishment(s.DB, model.SuspendPunishment))
				jobPostRoute.POST("", jobPostController.CreateJobPostHandler)

			}

			// Reporting endpoints
			reportRoute := needAuth.Group("/report")
			{
				reportRoute.PUT("/:type/:id", middleware.CheckRole(model.RoleAdmin), reportController.UpdateReportStatus)
				reportRoute.GET("", middleware.CheckRole(model.RoleAdmin), reportController.GetReport)
				reportRoute.POST("/user", reportController.CreateUserReport)
				reportRoute.POST("/post", middleware.CheckRole(model.RoleCPSK, model.RoleVisitor), reportController.CreatePostReport)
			}

			needCompanyAdmin := needAuth.Group("")
			{
				needCompanyAdmin.Use(middleware.CheckRole(model.RoleAdmin, model.RoleCompany))
				needCompanyAdmin.PATCH("jobpost/:id", jobPostController.EditJobPost)
				needCompanyAdmin.DELETE("jobpost/:id", jobPostController.DeleteJobPost)
			}

			needAdmin := needAuth.Group("")
			{
				needAdmin.Use(middleware.CheckRole(model.RoleAdmin))
				needAdmin.GET("get-companies", adminController.GetCompanies)
				needAdmin.GET("get-cpsk", adminController.GetCPSK)
				needAdmin.GET("get-visitors", adminController.GetVisitors)
				needAdmin.PATCH("verify-company/:company_id", adminController.VerifyCompany)
				needAdmin.PUT("punish/:user_id", punishmentController.PunishUser)
				needAdmin.DELETE("punish/:user_id", punishmentController.DeletePunishmentRecord)
			}

			// CPSK routes: apply role check once for all CPSK endpoints
			needCPSK := needAuth.Group("")
			{
				needCPSK.Use(middleware.CheckRole(model.RoleCPSK))
				cpskRoute := needCPSK.Group("/cpsk")
				{
					cpskRoute.PATCH("profile", cpskController.EditCPSKProfile)
					cpskRoute.GET("myprofile", cpskController.GetMyCPSKProfile)
					cpskRoute.POST("profile/resume", middleware.SizeLimit(10<<20), fileController.UploadResume)
				}

				needCPSK.Use(middleware.CheckPunishment(s.DB, model.SuspendPunishment))
				needCPSK.POST("application", applicationController.ApplicationHandler)
			}

		}
	}

	r.GET("/swagger/*any", middleware.RateLimiterMiddleware(uint(30)),ginSwagger.WrapHandler(swaggerFiles.Handler))

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
