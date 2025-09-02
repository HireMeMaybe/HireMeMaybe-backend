// Package database implement connection to database service and initialize ORM.
package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	// It's something abt database I don't know 😭
	_ "github.com/jackc/pgx/v5/stdlib"
	// Load .env file to environments
	_ "github.com/joho/godotenv/autoload"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"HireMeMaybe-backend/internal/model"
)

// Service represents a service that interacts with a database.
// type Service interface {
// 	// Health returns a map of health status information.
// 	// The keys and values in the map are service-specific.
// 	Health() map[string]string

// 	// Close terminates the database connection.

// 	// It returns an error if the connection cannot be closed.
// 	Close() error

// 	// Migrate database
// 	Migrate() error

// 	// Get ORM object
// 	GetORM() *gorm.DB
// }

// Service contain sql.DB instance and gorm instance

var (
	database   = os.Getenv("DB_DATABASE")
	password   = os.Getenv("DB_PASSWORD")
	username   = os.Getenv("DB_USERNAME")
	port       = os.Getenv("DB_PORT")
	host       = os.Getenv("DB_HOST")
	// DBinstance is instance or GORM orm as an interface to database
	DBinstance *gorm.DB
)

// InitializeDatabase constrct new Database service with ORM
func InitializeDatabase() error {
	// Reuse Connection
	if DBinstance != nil {
		return nil
	}

	useEnvConnStr, err := strconv.ParseBool(os.Getenv("USE_CONNECTION_STR")); 
	if err != nil {
		log.Fatal("USE_CONNECTION_STR environments variables are invalid")
	}

	var connStr string
	if useEnvConnStr {
		connStr = os.Getenv("DB_CONNECTION_STR")
	} else {
		connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", username, password, host, port, database)
	}

	DBinstance, err = gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		return err
	}

	if err := Migrate(); err != nil {
		return err
	}

	return nil
}

// Migrate database
func Migrate() error {
	err := DBinstance.AutoMigrate(model.MigrateAble...)
	if err != nil {
		return err
	}
	return nil
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	oriDB, err := DBinstance.DB()
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Printf("db down: %v", err) // Log the error and terminate the program
		return stats
	}

	// Ping the database
	err = oriDB.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Printf("db down: %v", err) // Log the error and terminate the program
		return stats
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := oriDB.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	// Evaluate stats to provide a health message
	if dbStats.OpenConnections > 40 { // Assuming 50 is the max for this example
		stats["message"] = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func Close() error {
	log.Printf("Disconnected from database: %s", database)
	oriDB, err := DBinstance.DB()
	if err != nil {
		return err
	}
	return oriDB.Close()
}
