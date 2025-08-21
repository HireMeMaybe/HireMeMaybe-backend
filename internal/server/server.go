package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	// Load env file into environments.
	_ "github.com/joho/godotenv/autoload"

	"HireMeMaybe-backend/internal/database"
)

// Server contain port which server are running on and database instance
type Server struct {
	port int

	db *database.Service
}

// NewServer construct new Server instance
func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))

	if err := database.InitializeDatabase(); err != nil {
		log.Fatalf("Database failed to initialized: %s", err)
	}
	
	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
