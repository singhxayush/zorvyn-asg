package main

import (
	"context"
	"log"
	"os"

	"zorvyn-asg/internal/database"

	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	db := database.New()
	defer db.Close()

	email := os.Getenv("ADMIN_EMAIL")
	password := os.Getenv("ADMIN_PASSWORD")
	username := os.Getenv("ADMIN_USERNAME")

	if email == "" || password == "" {
		log.Fatal("ADMIN_EMAIL and ADMIN_PASSWORD must be set in .env")
	}

	// Check if admin already exists
	_, err := db.GetUserByEmail(context.Background(), email)
	if err == nil {
		log.Println("Admin user already exists. Exiting.")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	adminUser := &database.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         "admin",
		Status:       "active",
	}

	if err := db.CreateUser(context.Background(), adminUser); err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}

	log.Println("Successfully created initial Admin user!")
}
