package routes

import (
	"github.com/amrrdev/trawl/services/auth/internal/handler"
	"github.com/amrrdev/trawl/services/shared/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, authHandlers *handler.AuthHandler, authMiddleware *middleware.AuthMiddleware) {
	auth := router.Group("/auth")
	{
		// Public routes - no authentication required
		auth.POST("/register", authHandlers.Register)
		auth.POST("/login", authHandlers.Login)
	}

	// Protected routes - authentication required
	protected := router.Group("/protected")
	protected.Use(authMiddleware.RequireAuth())
	{
		// Example: Add protected endpoints here
		// protected.GET("/profile", profileHandler.GetProfile)
		// protected.PUT("/profile", profileHandler.UpdateProfile)
	}
}
