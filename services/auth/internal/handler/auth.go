package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/amrrdev/trawl/services/auth/internal/services"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type RegisterBody struct {
	Name     string `json:"name" binding:"required,min=2"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	body := &RegisterBody{}

	if err := c.ShouldBindJSON(body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	resp, err := h.authService.Register(c, body.Name, body.Email, body.Password)
	if err != nil {
		fmt.Println(err)
		statusCode := http.StatusInternalServerError
		message := "Failed to register user"

		errMsg := err.Error()
		if strings.Contains(errMsg, "already exists") {
			statusCode = http.StatusConflict
			message = "User already exists"
		}

		c.JSON(statusCode, gin.H{
			"error": message,
		})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

type LoginBody struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	body := &LoginBody{}

	if err := c.ShouldBindJSON(body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	resp, err := h.authService.Login(c, body.Email, body.Password)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "Login failed"

		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid credentials") {
			statusCode = http.StatusUnauthorized
			message = "Invalid credentials"
		} else if strings.Contains(errMsg, "deactivated") {
			statusCode = http.StatusForbidden
			message = "Account is deactivated"
		}

		c.JSON(statusCode, gin.H{
			"error": message,
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
