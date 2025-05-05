package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"gitlab-orchestrator-back/internal/logger"
	"os"
	"strconv"
)

// Configuration holds all application config values
type Configuration struct {
	// Server settings
	Port string `env:"PORT" default:"8080"`

	// Database settings
	DBHost     string `env:"DB_HOST" default:"localhost"`
	DBUser     string `env:"DB_USER" default:"postgres"`
	DBPassword string `env:"DB_PASSWORD" default:"postgres"`
	DBName     string `env:"DB_NAME" default:"demo_dispatcher"`
	DBPort     string `env:"DB_PORT" default:"5432"`

	// GitLab settings
	GitlabAPIURL               string `env:"GITLAB_API_URL"`
	GitlabToken                string `env:"GITLAB_TOKEN"`
	GitlabProjectID            int    `env:"GITLAB_PROJECT_ID"`
	GitlabTriggerPipelineToken string `env:"GITLAB_TRIGGER_PIPELINE_TOKEN"`

	// Logger settings
	LogLevel string `env:"LOG_LEVEL" default:"info"`
}

// Global configuration instance
var Config = &Configuration{}

// Load initializes configuration from environment variables
func (c *Configuration) Load() error {

	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		logger.WarnWithCaller("No .env file found, using environment variables")
	} else {
		logger.InfoWithCaller(".env file loaded successfully")
	}

	// Load server settings
	c.Port = getEnvWithDefault("PORT", "8080")

	// Load database settings
	c.DBHost = getEnvWithDefault("DB_HOST", "localhost")
	c.DBUser = getEnvWithDefault("DB_USER", "postgres")
	c.DBPassword = getEnvWithDefault("DB_PASSWORD", "postgres")
	c.DBName = getEnvWithDefault("DB_NAME", "demo_dispatcher")
	c.DBPort = getEnvWithDefault("DB_PORT", "5432")

	// Load GitLab settings
	c.GitlabAPIURL = os.Getenv("GITLAB_API_URL")
	c.GitlabToken = os.Getenv("GITLAB_TOKEN")

	// Parse GitLab Project ID
	projectIDStr := os.Getenv("GITLAB_PROJECT_ID")
	if projectIDStr != "" {
		projectID, err := strconv.Atoi(projectIDStr)
		if err != nil {
			return fmt.Errorf("invalid GITLAB_PROJECT_ID: %v", err)
		}
		c.GitlabProjectID = projectID
	}

	c.GitlabTriggerPipelineToken = os.Getenv("GITLAB_TRIGGER_PIPELINE_TOKEN")

	// Load logger settings
	c.LogLevel = getEnvWithDefault("LOG_LEVEL", "info")

	// Validate configuration
	return c.validate()
}

// validate ensures all required configuration is present and valid
func (c *Configuration) validate() error {
	// Validate GitLab settings (required for core functionality)
	if c.GitlabAPIURL == "" {
		return fmt.Errorf("GITLAB_API_URL is required")
	}

	if c.GitlabToken == "" {
		return fmt.Errorf("GITLAB_TOKEN is required")
	}

	if c.GitlabProjectID == 0 {
		return fmt.Errorf("GITLAB_PROJECT_ID is required and must be a valid integer")
	}

	if c.GitlabTriggerPipelineToken == "" {
		return fmt.Errorf("GITLAB_TRIGGER_PIPELINE_TOKEN is required")
	}

	// Validate database settings
	if c.DBHost == "" || c.DBUser == "" || c.DBName == "" {
		return fmt.Errorf("database configuration is incomplete")
	}

	return nil
}

// GetDatabaseDSN returns the connection string for PostgreSQL
func (c *Configuration) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Europe/Moscow",
		c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort,
	)
}

// GetLogLevel returns the configured log level
func (c *Configuration) GetLogLevel() string {
	return c.LogLevel
}

// Print outputs the current configuration for debugging
func (c *Configuration) Print() {
	logger.InfoWithCaller("Configuration:")
	logger.InfofWithCaller("- Server Port: %s", c.Port)
	logger.InfofWithCaller("- Database: %s@%s:%s/%s", c.DBUser, c.DBHost, c.DBPort, c.DBName)
	logger.InfofWithCaller("- GitLab API URL: %s", c.GitlabAPIURL)
	logger.InfofWithCaller("- GitLab Project ID: %d", c.GitlabProjectID)
	logger.InfofWithCaller("- Log Level: %s", c.LogLevel)

	// Don't log sensitive information
	if c.GitlabToken != "" {
		logger.InfoWithCaller("- GitLab Token: [CONFIGURED]")
	} else {
		logger.InfoWithCaller("- GitLab Token: [NOT CONFIGURED]")
	}

	if c.GitlabTriggerPipelineToken != "" {
		logger.InfoWithCaller("- GitLab Pipeline Token: [CONFIGURED]")
	} else {
		logger.InfoWithCaller("- GitLab Pipeline Token: [NOT CONFIGURED]")
	}
}

// Helper function to get environment variable with a default value
func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
