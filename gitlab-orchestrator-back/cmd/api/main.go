package main

import (
	"gitlab-orchestrator-back/internal/config"
	"gitlab-orchestrator-back/internal/database"
	"gitlab-orchestrator-back/internal/gitlab"
	"gitlab-orchestrator-back/internal/handlers"
	"gitlab-orchestrator-back/internal/logger"
	"gitlab-orchestrator-back/internal/middleware"
	"gitlab-orchestrator-back/internal/routes"
	"gitlab-orchestrator-back/internal/scheduler"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

func main() {
	// Initialize logger
	logger.Init()
	logger.InfoWithCaller("Starting gitlab-orchestrator application")

	// Load configuration
	if err := config.Config.Load(); err != nil {
		logger.FatalfWithCaller("Failed to load configuration: %v", err)
	}

	// Print configuration
	config.Config.Print()

	// Initialize database
	if err := database.InitDB(); err != nil {
		logger.FatalfWithCaller("Failed to initialize database: %v", err)
	}

	// Create GitLab client using configuration
	gitClient := gitlab.NewClient()

	// Start scheduler with configured client
	logger.InfoWithCaller("Starting task scheduler")
	scheduler.StartRunnerScheduler(gitClient)

	// Create a new Echo instance
	e := echo.New()
	logger.InfoWithCaller("Initializing HTTP server (Echo)")

	// Middleware
	e.Use(echoMiddleware.Logger())
	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.CORS())
	e.Use(middleware.RequestIDMiddleware)
	logger.InfoWithCaller("Middleware configured")

	// Setup routes
	routes.SetupRoutes(e, &handlers.Handler{})
	logger.InfoWithCaller("API routes configured")

	// Start server
	logger.InfofWithCaller("Starting HTTP server on port %s", config.Config.Port)
	if err := e.Start(":" + config.Config.Port); err != nil {
		logger.FatalfWithCaller("Error starting server: %v", err)
	}
}
