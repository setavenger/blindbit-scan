package server

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/internal/daemon"
	"github.com/setavenger/blindbit-scan/pkg/logging"
	"github.com/setavenger/blindbit-scan/pkg/networking/nwc"
	"github.com/setavenger/blindbit-scan/pkg/utils"
)

func StartNewServer(
	d *daemon.Daemon,
	nip47Controller *nwc.Nip47Controller,
) error {
	s := Server{
		Daemon:          d,
		Nip47Controller: nip47Controller,
	}
	return s.RunServer()
}

type Server struct {
	Daemon          *daemon.Daemon
	Nip47Controller *nwc.Nip47Controller
}

func (s *Server) RunServer() error {
	router := gin.Default()
	// router := gin.New()
	// router.Use(gin.Recovery())
	// router.Use(gin.Logger())

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "PUT"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// we only allow this in simple mode
	// too much complexity in private mode
	if !config.PrivateMode {
		router.PUT("/new-keys", s.PutSilentPaymentKeys)
	}

	// setup instance no auth
	if config.PrivateMode {
		// setup instance
		router.POST("/setup-instance", func(ctx *gin.Context) {
			authExists := utils.CheckIfFileExists(config.PathDbAuth)
			walletExists := utils.CheckIfFileExists(config.PathDbWallet)

			if authExists || walletExists {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": "instance already (partially) setup"})
				ctx.Abort()
				return
			}
			ctx.Next()
		}, s.SetupInstance)

		// unlock the wallet
		// this is used to unlock the wallet when the user has already setup the instance
		// no basic auth because basic auth is decrypted with the password
		router.POST("/unlock", s.Unlock)
	}

	router.Use(s.basicAuthMiddleware())

	// BlindBit adaptation of Nostr Wallet Connect
	router.POST("/new-nwc-connection", s.NewNwcConnection)

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

	walletReadyGroup.POST("/rescan", s.PostRescan)

	if err := router.Run(config.ExposeHttpHost); err != nil {
		logging.L.Err(err).Msg("failed to start server")
		return err
	}
	return nil
}
