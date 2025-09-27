package database

import (
	"context"
	"log"
	"testing"

	// Load env
	_ "github.com/joho/godotenv/autoload"
)

func TestMain(m *testing.M) {
	teardown, err := StartTestDB()
	if err != nil {
		log.Fatalf("could not start postgres container: %v", err)
	}

	m.Run()

	if teardown != nil && teardown(context.Background()) != nil {
		log.Fatalf("could not teardown postgres container: %v", err)
	}
}

func TestNew(t *testing.T) {
	_, err := NewDBInstance()
	if err != nil {
		t.Fatalf("Database failed to initialize: %s", err)
	}
}

func TestHealth(t *testing.T) {
	db, err := NewDBInstance()
	if err != nil {
		t.Fatalf("Database failed to initialize: %s", err)
	}
	stats := db.Health()

	if stats["status"] != "up" {
		t.Fatalf("expected status to be up, got %s", stats["status"])
	}

	if _, ok := stats["error"]; ok {
		t.Fatalf("expected error not to be present")
	}

	if stats["message"] != "It's healthy" {
		t.Fatalf("expected message to be 'It's healthy', got %s", stats["message"])
	}
}

func TestClose(t *testing.T) {
	db, err := NewDBInstance()
	if err != nil {
		t.Fatalf("Database failed to initialize: %s", err)
	}

	if db.Close() != nil {
		t.Fatalf("expected Close() to return nil")
	}
}
