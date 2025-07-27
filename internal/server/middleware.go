package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/setavenger/blindbit-scan/internal"
	"github.com/setavenger/blindbit-scan/internal/config"
)

// internal/server/middleware.go
func (s *Server) basicAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		status := internal.InstanceStatus()
		switch status {
		case internal.StatusSetupRequired:
			c.JSON(http.StatusConflict, gin.H{"error": "Wallet not setup yet"})
			c.Abort()
			return
		case internal.StatusLocked:
			c.JSON(http.StatusLocked, gin.H{"error": "Wallet not unlocked"})
			c.Abort()
			return
		default:
			creds := config.GetAuthCredentials()
			if creds == nil {
				// this should not happen but just putting this here
				c.JSON(http.StatusInternalServerError, gin.H{"error": "bad creds"})
				c.Abort()
				return
			}

			username, password, ok := c.Request.BasicAuth()
			if !ok || username != creds.Username || password != creds.Password {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
