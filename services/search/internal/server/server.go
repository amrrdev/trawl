package server

import (
	"github.com/amrrdev/trawl/services/search/internal/handler"
	"github.com/amrrdev/trawl/services/search/internal/routes"
	"github.com/amrrdev/trawl/services/shared/middleware"
	"github.com/gin-gonic/gin"
)

func NewServer(searchHandler *handler.SearchHandler, authMiddleware *middleware.AuthMiddleware) *gin.Engine {
	g := gin.Default()
	api := g.Group("/api/v1")
	routes.RegisterRoutes(api, searchHandler, authMiddleware)
	return g
}
