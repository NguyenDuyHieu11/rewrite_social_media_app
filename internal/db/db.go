package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres" // Or gorm.io/driver/mysql, gorm.io/driver/sqlserver
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	// Adjust connection string for your database engine
	dsn := "host=localhost user=postgres password=yourpassword dbname=social_app_db port=5432 sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	fmt.Println("Database connection established.")
	return db
}
