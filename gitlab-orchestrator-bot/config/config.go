package config

import (
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

const (
	BtnAddStand        = "btnAddStand"
	BtnTest            = "btnTest"
	BtnDone            = "btnDone"
	BtnCancel          = "btnCancel"
	BtnDoneStep2       = "btnDoneStep2"
	NumberOfLinesSubos = 2

	StartMessage = "Добро пожаловать в Aggregator Bot!\nВ настоящее время доступны команды:\n1. /createstand"
)

var (
	AllowedUsers  = make(map[int64]bool)
	AllowedAdmins = make(map[int64]bool)
	Config        = &Configuration{}
	UserStates    = make(map[int64]*UserContext)
	UserStands    = make(map[int64][]StandData)
	AdminStands   = make([]StandData, 0)
)

type StandData struct {
	NameStand string   `json:"nameStand"`
	Products  []string `json:"products"`
	UserID    int64    `json:"userID"`
	Ref       string   `json:"ref"`
}

type UserContext struct {
	//vars
	FilterSubos     map[string]bool
	CreateStandName string

	//states
	WaitingForMessageStand    bool
	WaitingApproveCreateStand bool
}

type Configuration struct {
	// API Configuration
	BackendURL string `env:"BACKEND_URL"`
	BotToken   string `env:"TOKEN"`

	// Application Configuration
	LogLevel string `env:"LOG_LEVEL" default:"info"`
	Env      string `env:"ENV" default:"development"`
	Domain   string `env:"DOMAIN" default:".example.com"`

	// Internal
	mu sync.Mutex
}

// LoadEnvFile attempts to load environment variables from a .env file
func LoadEnvFile(file string) error {
	if file == "" {
		file = ".env"
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		log.Printf("Warning: %s file not found, using environment variables", file)
		return nil
	}

	return godotenv.Load(file)
}

// Inits initializes the configuration with environment variables
func (c *Configuration) Inits() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Try to load .env file first
	LoadEnvFile("")

	// Load required configuration
	c.BotToken = os.Getenv("TOKEN")
	if c.BotToken == "" {
		log.Fatalf("TOKEN is not set. This is required for the Telegram bot to function.")
	}

	c.BackendURL = os.Getenv("BACKEND_URL")
	if c.BackendURL == "" {
		log.Fatalf("BACKEND_URL is not set. This is required to connect to the backend API.")
	}

	// Load optional configuration with defaults
	c.LogLevel = getEnvWithDefault("LOG_LEVEL", "info")
	c.Env = getEnvWithDefault("ENV", "development")

	// Override Domain constant if specified in environment
	envDomain := os.Getenv("DOMAIN")
	if envDomain != "" {
		c.Domain = envDomain
	} else {
		c.Domain = ".example.com" // Use the constant as default
	}

	// Validate configuration
	c.validate()
}

// validate checks that the configuration is valid
func (c *Configuration) validate() {
	// Check that BackendURL is a valid URL format (simple check)
	if c.BackendURL == "" || (len(c.BackendURL) < 8) {
		log.Fatalf("BACKEND_URL is not a valid URL: %s", c.BackendURL)
	}

	// Check that BotToken is provided and has minimal length
	if len(c.BotToken) < 10 {
		log.Fatalf("TOKEN appears to be invalid: too short")
	}

	// Check that environment is valid
	validEnvs := map[string]bool{"development": true, "staging": true, "production": true}
	if !validEnvs[c.Env] {
		log.Printf("Warning: ENV '%s' is not a recognized environment (development, staging, production)", c.Env)
	}
}

// Print outputs the current configuration for debugging
func (c *Configuration) Print() {
	log.Printf("Configuration:")
	log.Printf("- Environment: %s", c.Env)
	log.Printf("- Log Level: %s", c.LogLevel)
	log.Printf("- Backend URL: %s", c.BackendURL)
	log.Printf("- Domain: %s", c.Domain)

	// Don't print the token for security reasons
	log.Printf("- Bot Token: %s******", c.BotToken[:4])
}

// Helper function to get environment variable with a default value
func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
