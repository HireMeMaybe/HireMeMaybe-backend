package main

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
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

	database.InitializeDatabase()

	// Connect to database (SQLite for simplicity)
	db := database.DBinstance

	// Generate unique username and password
	username := generateUniqueUsername(db)
	password := generateRandomString(8)

	// Hash the password before storing
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("failed to hash password: ", err)
	}

	// Create admin user
	admin := model.User{
		Username: username, 
		Password: string(hashedPassword),
		Role: model.RoleAdmin,
	}
	if err := db.Create(&admin).Error; err != nil {
		log.Fatal("failed to create admin: ", err)
	}

	// Print credentials (only show plain password here!)
	fmt.Println("Admin credentials generated successfully!")
	fmt.Println("======================================")
	fmt.Printf("Username: %s\n", admin.Username)
	fmt.Printf("Password: %s\n", password)
	fmt.Println("======================================")

	os.Exit(0)
}
