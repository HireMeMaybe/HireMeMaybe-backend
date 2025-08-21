// Package server contain implementation of go-gin-server and each route handlers
package server

import (
	"HireMeMaybe-backend/internal/database"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes will register each http endpoint routes to bound Server instance
func RegisterRoutes() http.Handler {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // Add your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	r.GET("/", HelloWorldHandler)

	r.GET("/health", healthHandler)

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
