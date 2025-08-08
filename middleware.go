package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// 中间件：API Key 认证
func authMiddleware() gin.HandlerFunc {
	token := os.Getenv("API_TOKEN")
	return func(c *gin.Context) {
		if len(token) == 0 {
			c.Next()
			return
		}
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing API key"})
			c.Abort()
			return
		}
		if authHeader != token {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}
		c.Next()
	}
}
