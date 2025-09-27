package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	// Load env
	_ "github.com/joho/godotenv/autoload"
)

func StartTestDB() (func(context.Context, ...testcontainers.TerminateOption) error, error) {
	var (
		dbName = "database"
		dbPwd  = "password"
		dbUser = "user"
		dbport = "5432"
	)

	dbContainer, err := postgres.Run(
		context.Background(),
		"postgres:latest",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPwd),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		return nil, err
	}

	database = dbName
	password = dbPwd
	username = dbUser
	port = dbport
	useEnvConnStr = "false"

	dbHost, err := dbContainer.Host(context.Background())
	if err != nil {
		return dbContainer.Terminate, err
	}

	dbPort, err := dbContainer.MappedPort(context.Background(), nat.Port("5432/tcp"))
	if err != nil {
		return dbContainer.Terminate, err
	}

	host = dbHost
	port = dbPort.Port()

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort.Port(), dbUser, dbPwd, dbName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return dbContainer.Terminate, err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Fatal("Fail to close database")
		}
	}()

	_, err = db.ExecContext(context.Background(), fmt.Sprintf(`CREATE EXTENSION IF NOT EXISTS "%s";`, "uuid-ossp"))
	if err != nil {
		return dbContainer.Terminate, err
	}

	return dbContainer.Terminate, err
}
