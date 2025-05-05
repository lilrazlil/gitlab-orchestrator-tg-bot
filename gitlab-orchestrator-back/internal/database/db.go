package database

import (
	"gitlab-orchestrator-back/internal/config"
	"gitlab-orchestrator-back/internal/logger"
	"gitlab-orchestrator-back/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDB initializes the database connection
func InitDB() error {
	var err error

	// Get DSN from configuration
	dsn := config.Config.GetDatabaseDSN()

	logger.InfoWithCaller("Connecting to database...")

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.ErrorfWithCaller("Failed to connect to database: %v", err)
		return err
	}

	logger.InfoWithCaller("Connected to database successfully")

	// Auto migrate the database schema
	err = DB.AutoMigrate(
		&models.User{},      // Correct struct for the user table
		&models.Stand{},     // Struct for the stand table
		&models.Step{},      // Struct for the step table
		&models.Pipeline{},  // Struct for the pipeline table
		&models.Job{},       // Struct for the job table
		&models.Subos{},     // Struct for the subos table
		&models.StepState{}, // Struct for the step state table
	)
	if err != nil {
		logger.ErrorfWithCaller("Failed to migrate database: %v", err)
		return err
	}

	logger.InfoWithCaller("Database migration completed successfully")
	return nil
}
