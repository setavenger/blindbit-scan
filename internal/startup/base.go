package startup

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/internal/daemon"
	nwcserver "github.com/setavenger/blindbit-scan/internal/nwc_server"
	"github.com/setavenger/blindbit-scan/internal/server"
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

	flag.BoolVar(&config.PrivateMode, "private", false, "BlindBit Scan will run in private mode. All data on disk will be encrypted all data will only be decrypted in memory. Upon restart the unlock endpoint needs to be called to decrypt data and start the scanning.")

	flag.Parse()
}

func RunProgram() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	var err error

	err = config.SetupConfigs(config.DirectoryPath)
	if err != nil {
		logging.L.Panic().Err(err).
			Msg("startup failed, could not setup configs")
	}

	var d *daemon.Daemon

	if config.PrivateMode {
		logging.L.Info().Msg("running in privacy mode")
		logging.L.Info().Msg("privacy mode requires a setup api call to setup the instance")
		d, err = StartupWithPrivateMode()
	} else {
		d, err = StartupWithSimpleMode()
	}

	if err != nil {
		logging.L.Panic().Err(err).
			Msg("startup failed, could not setup daemon")
	}

	// Setup BlindBit Nostr Wallet Connect
	nwcServer := nwcserver.NewNwcServer(d)

	logging.L.Info().Msg("attempting to load NWC apps from disk")
	controller, err := d.DBWriter.TryLoadingControllerFromDisk(context.Background(), config.PathDbNWC)
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
		// if the wallet is not setup we wait for the keys ready signal
		if d.Wallet == nil {
			logging.L.Info().Msg("waiting for keys")
			<-config.KeysReadyChan
			err = d.SetupExternalClients()
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
