package server

import (
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
	router := gin.New()
	router.GET("/height", s.GetCurrentHeight)
	return router.Run(config.ExposeHttpHost)
}
