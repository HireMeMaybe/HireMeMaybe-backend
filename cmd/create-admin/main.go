// Package main for creating admin cli tool
package main

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"gorm.io/gorm"
)

// generateRandomString creates a random hex string of length n
func generateRandomString(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(bytes)
}

// generateUniqueUsername tries until a unique username is found
func generateUniqueUsername(db *gorm.DB) string {
	for {
		username := "admin_" + generateRandomString(4)
		var count int64
		db.Model(&model.User{}).Where("username = ?", username).Count(&count)
		if count == 0 {
			return username
		}
		// If username exists, loop again
	}
}

func main() {

	db, err := database.GetMainDB()
	if err != nil {
		log.Fatal("Fail to initialize database: ", err)
	}

	// Generate unique username and password
	username := generateUniqueUsername(db.DB)
	password := generateRandomString(8)

	// Hash the password before storing
	utilities.CreateAdmin(password, username, db.DB)

	// Print credentials (only show plain password here!)
	fmt.Println("Admin credentials generated successfully!")
	fmt.Println("======================================")
	fmt.Printf("Username: %s\n", username)
	fmt.Printf("Password: %s\n", password)
	fmt.Println("======================================")

	os.Exit(0)
}
