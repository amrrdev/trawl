package server

import (
	"github.com/amrrdev/trawl/services/indexing/internal/handler"
	"github.com/amrrdev/trawl/services/indexing/internal/routes"
	"github.com/amrrdev/trawl/services/shared/middleware"
	"github.com/gin-gonic/gin"
)

func NewServer(documentHandler *handler.DocumentHandler, authMiddleware *middleware.AuthMiddleware) *gin.Engine {
	g := gin.Default()
	api := g.Group("/api/v1")
	routes.RegisterRoutes(api, documentHandler, authMiddleware)
	return g
}
