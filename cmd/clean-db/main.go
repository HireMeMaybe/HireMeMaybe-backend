// Command-line tool to clean the database by dropping all tables in the public schema.
package main

import (
	"HireMeMaybe-backend/internal/database"
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {

	// Warning message
	fmt.Println("⚠️ WARNING: This command will DROP ALL TABLES in the 'public' schema of your database.")
	fmt.Println("This action is irreversible. Do you want to continue? (yes/no): ")

	// Ask for confirmation
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read input: %v", err)
	}
	input = strings.TrimSpace(strings.ToLower(input))

	if input != "yes" {
		fmt.Println("Operation cancelled.")
		return
	}

	// Initialize database and check for errors
	if err := database.InitializeDatabase(); err != nil {
		log.Fatalf("Database failed to initialize: %v", err)
	}

	if database.DBinstance == nil {
		log.Fatalf("Database instance is nil after initialization")
	}

	// SQL command to drop all tables
	sql := `
	DO $$ 
		DECLARE 
			r RECORD;
		BEGIN 
			FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
				EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
			END LOOP; 
		END $$;
	`

	// Execute raw SQL
	if err := database.DBinstance.Exec(sql).Error; err != nil {
		log.Fatalf("failed to execute drop command: %v", err)
	}

	fmt.Println("✅ All tables dropped successfully.")
}
