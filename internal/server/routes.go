package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) GetCurrentHeight(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"height": s.Daemon.Wallet.LastScanHeight})
}
