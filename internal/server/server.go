package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"HireMeMaybe-backend/internal/database"
)

// MyServer is a struct that holds the server configuration and dependencies.
type MyServer struct {
	DB   *database.DBinstanceStruct
	port int
}

// NewServer construct new Server instance
func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))

	db, err := database.GetMainDB()
	// Initialize database and check for errors
	if err != nil {
		log.Fatalf("Database failed to initialized: %s", err)
	}

	// Declare Server config
	myServer := &MyServer{
		DB:   db,
		port: port,
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      myServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
