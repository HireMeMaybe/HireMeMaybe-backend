package main

import (
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {

	fmt.Println("Generating admin account")

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Enter password: ")
	password1, _ := reader.ReadString('\n')
	password1 = strings.TrimSpace(password1)

	fmt.Print("Confirm password: ")
	password2, _ := reader.ReadString('\n')
	password2 = strings.TrimSpace(password2)

	if password1 == password2 {
		fmt.Printf("Username: %s\nPassword confirmed.\n", username)
	} else {
		fmt.Println("Passwords do not match.")
		os.Exit(0)
	}

	if err := database.InitializeDatabase(); err != nil {
		fmt.Println("Failed to initialize database %w", err)
		os.Exit(1)
	}

	db := database.DBinstance

	admin := model.User{}

	if err := db.Where("id = ?", username).First(&admin).Error; err == nil {
		fmt.Println("Username already taken")
	}

}