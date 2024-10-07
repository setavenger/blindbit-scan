package server

import (
	"log/slog"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/internal/daemon"
)

func StartNewServer(d *daemon.Daemon) error {
	s := Server{
		Daemon: d,
	}
	return s.RunServer()
}

type Server struct {
	Daemon *daemon.Daemon
}

func (s *Server) RunServer() error {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "PUT"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
		// ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		// AllowOriginFunc: func(origin string) bool {
		// return origin == "https://github.com"
		// },
	}))

	router.Use(gin.BasicAuth(gin.Accounts{
		config.AuthUser: config.AuthPass,
	}))

	router.PUT("/new-keys", s.PutSilentPaymentKeys)

	// the wallet has to be set up to reach these endpoints to avoid crashes
	walletReadyGroup := router.Group("/")

	walletReadyGroup.Use(func(c *gin.Context) {
		if s.Daemon.Wallet == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "wallet not ready"})
			c.Abort()
			return
		}
	})

	walletReadyGroup.GET("/height", s.GetCurrentHeight)
	walletReadyGroup.GET("/utxos", s.GetUtxos)
	walletReadyGroup.GET("/address", s.GetAddress)

	if err := router.Run(config.ExposeHttpHost); err != nil {
		slog.Error(err.Error())
		return err
	}
	return nil
}
