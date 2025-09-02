// Package server contain implementation of go-gin-server and each route handlers
package server

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/controller"
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/middleware"
	"HireMeMaybe-backend/internal/model"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
	r.GET("/needauth", middleware.RequireAuth(), thisNeedAuth)
	r.GET("/health", healthHandler)

	r.POST("/auth/google/cpsk", auth.CPSKGoogleLoginHandler)

	r.POST("/auth/google/company", auth.CompanyGoogleLoginHandler)

	r.GET("/auth/google/callback", auth.Callback)

	r.PUT("/cpsk/profile", middleware.RequireAuth(), controller.EditCPSKProfile)
	r.GET("/cpsk/myprofile", middleware.RequireAuth(), controller.GetMyCPSKProfile)
	r.POST("/cpsk/profile/resume", middleware.RequireAuth(), controller.UploadResume)

	r.GET("/file/:id", controller.GetFile)

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

func thisNeedAuth(c *gin.Context) {

	u, _ := c.Get("user")
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User information not provided",
		})
		return
	}

	user, ok := u.(model.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to assert type",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Welcome user %s", user.ID),
	})
}
