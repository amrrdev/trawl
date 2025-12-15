package routes

import (
	"github.com/amrrdev/trawl/services/auth/internal/handler"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, authHandlers *handler.AuthHandler) {
	g := router.Group("/auth")
	g.POST("/register", authHandlers.Register)
	g.POST("/login", authHandlers.Login)
}
