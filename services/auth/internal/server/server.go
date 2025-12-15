package server

import (
	"github.com/amrrdev/trawl/services/auth/internal/handler"
	"github.com/amrrdev/trawl/services/auth/internal/routes"
	"github.com/amrrdev/trawl/services/shared/middleware"
	"github.com/gin-gonic/gin"
)

func NewServer(authHandlers *handler.AuthHandler, authMiddleware *middleware.AuthMiddleware) *gin.Engine {
	g := gin.Default()
	api := g.Group("/api/v1")
	routes.RegisterRoutes(api, authHandlers, authMiddleware)

	return g
}
