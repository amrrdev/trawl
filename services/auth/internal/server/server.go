package server

import (
	"github.com/amrrdev/trawl/services/auth/internal/handler"
	"github.com/amrrdev/trawl/services/auth/internal/routes"
	"github.com/gin-gonic/gin"
)

func NewServer(authHandlers *handler.AuthHandler) *gin.Engine {
	g := gin.Default()
	api := g.Group("/api/v1")
	routes.RegisterRoutes(api, authHandlers)

	return g
}
