package middleware

import "github.com/gin-gonic/gin"

// GetUserID extracts the authenticated user ID from the context
func GetUserID(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if !exists {
		return ""
	}
	return userID.(string)
}

// GetUserEmail extracts the authenticated user email from the context
func GetUserEmail(c *gin.Context) string {
	email, exists := c.Get("email")
	if !exists {
		return ""
	}
	return email.(string)
}
