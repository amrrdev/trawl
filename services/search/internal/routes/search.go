package routes

import (
	"github.com/amrrdev/trawl/services/search/internal/handler"
	"github.com/amrrdev/trawl/services/shared/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, searchHandler *handler.SearchHandler, authMiddleware *middleware.AuthMiddleware) {
	search := router.Group("/search")
	search.Use(authMiddleware.RequireAuth())
	{
		search.POST("", searchHandler.Search)
	}
}
