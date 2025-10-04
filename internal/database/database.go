// Package database implement connection to database service and initialize ORM.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	// It's something abt database I don't know 😭
	_ "github.com/jackc/pgx/v5/stdlib"
	// Load .env file to environments
	_ "github.com/joho/godotenv/autoload"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
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

// DBinstanceStruct is a struct that holds the GORM DB instance and related information.
type DBinstanceStruct struct {
	*gorm.DB
	// Config
	Config *DBConfig
	// cached raw DB and mutex for lazy-init
	sqlDB *sql.DB
	mu    sync.RWMutex
}

// DBConfig holds the configuration parameters for connecting to a database.
type DBConfig struct {
	Host      string
	Port      string
	User      string
	Password  string
	DBName    string
	Constr    string
	useConstr bool
}

func (d *DBConfig) getDsn() string {
	if d.useConstr {
		if d.Constr == "" {
			log.Fatal("DB_CONNECTION_STR is empty")
		}
		return d.Constr
	}
	if d.Host == "" || d.Port == "" || d.User == "" || d.Password == "" || d.DBName == "" {
		log.Fatal("Database configuration is incomplete")
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", d.User, d.Password, d.Host, d.Port, d.DBName)
}

var (
	database      = os.Getenv("DB_DATABASE")
	password      = os.Getenv("DB_PASSWORD")
	username      = os.Getenv("DB_USERNAME")
	port          = os.Getenv("DB_PORT")
	host          = os.Getenv("DB_HOST")
	useEnvConnStr = os.Getenv("USE_CONNECTION_STR")
	envConStr     = os.Getenv("DB_CONNECTION_STR")
	// DBinstance is instance or GORM orm as an interface to database
	dbInstance *DBinstanceStruct
)

// NewDBInstance creates a new DBinstanceStruct with the given configuration.
// It establishes a connection to the database and returns the instance or an error if the connection fails.
func NewDBInstance(config *DBConfig) (*DBinstanceStruct, error) {

	connStr := config.getDsn()

	gdb, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if gin.IsDebugging() {
		gdb = gdb.Debug()
	}

	newDb := &DBinstanceStruct{
		DB:     gdb,
		Config: config,
	}


	if err := newDb.installExtension(); err != nil {
		log.Fatal("failed to install extension: ", err)
	}
	if err := newDb.Migrate(); err != nil {
		log.Fatal("failed to migrate database: ", err)
	}
	newDb.createAdmin()

	return newDb, nil
}

// GetMainDB returns the main database instance, initializing it if necessary.
// It reads configuration from environment variables and ensures a single instance is used.
func GetMainDB() (*DBinstanceStruct, error) {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance, nil
	}

	useEnvConnStr, err := strconv.ParseBool(useEnvConnStr)
	if err != nil {
		log.Fatalf("USE_CONNECTION_STR environments variables are invalid %v", err)
	}

	config := &DBConfig{
		Host:      host,
		Port:      port,
		User:      username,
		Password:  password,
		DBName:    database,
		useConstr: useEnvConnStr,
		Constr:    envConStr,
	}

	return NewDBInstance(config)
}

// Raw returns the underlying *sql.DB, caching it after the first successful retrieval.
// It is safe for concurrent use.
func (d *DBinstanceStruct) Raw() (*sql.DB, error) {
	if d == nil {
		return nil, fmt.Errorf("DBinstanceStruct is nil")
	}

	// fast path: cached value
	d.mu.RLock()
	if d.sqlDB != nil {
		raw := d.sqlDB
		d.mu.RUnlock()
		return raw, nil
	}
	d.mu.RUnlock()

	// slow path: initialize
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.sqlDB != nil {
		return d.sqlDB, nil
	}
	if d.DB == nil {
		return nil, fmt.Errorf("gorm DB is nil")
	}
	raw, err := d.DB.DB()
	if err != nil {
		return nil, err
	}
	d.sqlDB = raw
	return raw, nil
}

func (d *DBinstanceStruct) createAdmin() {
	// Create admin user if not exist

	adminUsername := os.Getenv("ADMIN_USERNAME")
	adminPassword := os.Getenv("ADMIN_PASSWORD")

	if adminUsername == "" || adminPassword == "" {
		log.Println("Admin username or password not set, skipping admin creation")
		return
	}

	// Check if admin user already exists

	var count int64
	d.Model(&model.User{}).Where("role = ?", model.RoleAdmin).Count(&count)
	if count == 0 {
		utilities.CreateAdmin(adminPassword, adminUsername, d.DB)
	}

}

// Migrate database
func (d *DBinstanceStruct) Migrate() error {
	err := d.AutoMigrate(model.MigrateAble...)
	if err != nil {
		return err
	}
	return nil
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func (d *DBinstanceStruct) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	oriDB, err := d.Raw()
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
func (d *DBinstanceStruct) Close() error {
	log.Printf("Disconnected from database: %s", d.Config.Constr)
	oriDB, err := d.Raw()
	if err != nil {
		return err
	}
	return oriDB.Close()
}

func (d *DBinstanceStruct) installExtension() error {
	err := d.WithContext(context.Background()).Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`).Error
	if err != nil {
		return err
	}
	log.Println("uuid-ossp extension installed or already exists")
	return nil
}
