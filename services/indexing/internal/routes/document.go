package routes

import (
	"github.com/amrrdev/trawl/services/indexing/internal/handler"
	"github.com/amrrdev/trawl/services/shared/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, documentHandler *handler.DocumentHandler, authMiddleware *middleware.AuthMiddleware) {
	document := router.Group("/documents")
	document.Use(authMiddleware.RequireAuth())
	{
		document.POST("/upload-url/:filename", documentHandler.GetUploadUrl)
		document.POST("/download-url/:filename", documentHandler.GetDownloadUrl)
		document.GET("", documentHandler.ListFiles)
	}
}
