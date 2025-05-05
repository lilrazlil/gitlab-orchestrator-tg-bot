package routes

import (
	"gitlab-orchestrator-back/internal/handlers"

	"github.com/labstack/echo/v4"
)

// SetupRoutes configures all API routes for the application
func SetupRoutes(e *echo.Echo, h *handlers.Handler) {
	// API v1 group
	api := e.Group("/api/v1")

	// Health check
	api.GET("/health", h.HealthCheck)

	// User routes
	api.GET("/users", h.GetAllUsers)
	// api.GET("/users/:id", h.GetUserByID)
	// api.POST("/users", h.CreateUser)
	// api.PUT("/users/:id", h.UpdateUser)
	// api.DELETE("/users/:id", h.DeleteUser)

	// Subos (products) routes
	api.GET("/subos", h.GetSubos) // Simple map of code->name

	// Stand routes
	//api.POST("/stands/start", h.StartCreateStand)
	api.POST("/stands", h.CreateStand)
	api.GET("/stands", h.GetAllStands)
	// api.GET("/stands/:name/deployments", h.GetStandDeployments)

	// notify steps
	api.GET("/notify", h.GetNotification)
	api.POST("/notify", h.UpdateNotification)
}
