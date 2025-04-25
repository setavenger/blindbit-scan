package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/setavenger/blindbit-scan/internal/config"
)

// internal/server/middleware.go
func (s *Server) basicAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		creds := config.GetAuthCredentials()
		if creds == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Wallet not unlocked"})
			c.Abort()
			return
		}

		username, password, ok := c.Request.BasicAuth()
		if !ok || username != creds.Username || password != creds.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			c.Abort()
			return
		}
		c.Next()
	}
}
