package main

import (
	"bytes"
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/internal/daemon"
	nwcserver "github.com/setavenger/blindbit-scan/internal/nwc_server"
	"github.com/setavenger/blindbit-scan/internal/server"
	"github.com/setavenger/blindbit-scan/pkg/database"
	"github.com/setavenger/blindbit-scan/pkg/logging"
	"github.com/setavenger/blindbit-scan/pkg/networking/nwc"
)

func init() {
	// todo can this double reference work?
	flag.StringVar(
		&config.DirectoryPath,
		"datadir",
		config.DefaultDirectoryPath,
		"Set the base directory for blindbit-scan. Default directory is ~/.blindbit-scan.",
	)
	flag.Parse()
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	var err error

	// todo move to a go routine to avoid blocking
	err = config.SetupConfigs(config.DirectoryPath)
	if err != nil {
		logging.L.Panic().Err(err).
			Msg("startup failed, could not setup configs")
	}

	var d *daemon.Daemon

	d, err = daemon.SetupDaemonNoWallet()
	if err != nil {
		logging.L.Panic().Err(err).
			Msg("startup failed, could produce daemon hull")
	}
	w, err := database.TryLoadWalletFromDisk(config.PathDbWallet)
	if err != nil {
		logging.L.Warn().Err(err).
			Msg("startup failed, could setup full daemon")
	}
	d.Wallet = w

	// Setup BlindBit Nostr Wallet Connect
	nwcServer := nwcserver.NewNwcServer(d)

	logging.L.Info().Msg("attempting to load NWC apps from disk")
	controller, err := database.TryLoadingControllerFromDisk(context.Background(), config.PathDbNWC)
	if err != nil {
		logging.L.Panic().Err(err).Msg("failed to create new controller")
	}
	logging.L.Trace().Any("apps", controller.Apps()).Msg("controller data")

	controller.RegisterHandler(nwc.GET_INFO_METHOD, nwcServer.GetInfoHandler())
	controller.RegisterHandler(nwc.GET_BALANCE_METHOD, nwcServer.GetBalanceHandler())
	controller.RegisterHandler(nwc.LIST_UTXOS_METHOD, nwcServer.ListUtxosHandler())

	err = controller.ConnectRelay()
	if err != nil {
		logging.L.Panic().Err(err).Msg("failed to connect to relay")
	}
	go controller.StartListening()

	// http server
	go func() {
		err = server.StartNewServer(d, controller)
		if err != nil {
			logging.L.Panic().Err(err).
				Msg("startup failed, could start server")
		}
	}()

	// when we exit we still flush the last state
	defer d.SaveWalletToDB()

	go func() {
		// if the keys are not setup we wait
		if d.Wallet == nil || bytes.Equal(d.Wallet.SecretKeyScan[:], make([]byte, 32)) || bytes.Equal(d.Wallet.PubKeySpend[:], make([]byte, 33)) {
			logging.L.Info().Msg("waiting for keys")
			<-config.KeysReadyChan
			d, err = daemon.SetupDaemon(config.PathDbWallet)
			if err != nil {
				logging.L.Panic().Err(err).
					Msg("startup failed, could setup full daemon")
			}
		}
		go d.ContinuousScan()
	}()

	// wait for program stop signal
	<-interrupt
}
