package startup

import (
	"context"

	"github.com/setavenger/blindbit-scan/internal"
	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/internal/daemon"
	nwcserver "github.com/setavenger/blindbit-scan/internal/nwc_server"
	"github.com/setavenger/blindbit-scan/pkg/database"
	"github.com/setavenger/blindbit-scan/pkg/logging"
	"github.com/setavenger/blindbit-scan/pkg/networking/nwc"
	"github.com/setavenger/blindbit-scan/pkg/wallet"
)

func StartupWithPrivateMode(nip47Controller *nwc.Nip47Controller) (d *daemon.Daemon, err error) {
	walletExists := internal.CheckIfFileExists(config.PathDbWallet)
	authExists := internal.CheckIfFileExists(config.PathDbAuth)

	if walletExists && authExists {
		// we need to load the existing instance, wait for unlock api call
		logging.L.Info().Msg("Waiting for unlock request")
		d, err = SetupExistingInstancePrivateMode()
		if err != nil {
			logging.L.Panic().Err(err).
				Msg("startup failed, could produce daemon hull")
		}
	} else {
		// we need to setup a whole new instance and wait for Setup rest api call
		logging.L.Info().Msg("Waiting for setup-instance request")
		d, err = SetupNewInstancePrivateMode()
		if err != nil {
			logging.L.Panic().Err(err).
				Msg("startup failed, could produce daemon hull")
		}
	}

	// Setup BlindBit Nostr Wallet Connect
	nwcServer := nwcserver.NewNwcServer(d)

	var controller *nwc.Nip47Controller
	logging.L.Info().Msg("attempting to load NWC apps from disk")
	controller, err = d.DBWriter.TryLoadingControllerFromDisk(context.Background(), config.PathDbNWC)
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

	*nip47Controller = *controller

	return d, nil
}

func SetupExistingInstancePrivateMode() (d *daemon.Daemon, err error) {
	password := <-config.UnlockChan
	d, err = daemon.SetupDaemonNoWallet()
	if err != nil {
		logging.L.Panic().Err(err).
			Msg("startup failed, could only produce daemon hull")
	}

	d.SetDbWriter(&database.DBWriter{Password: password})

	w, err := d.DBWriter.LoadWalletFromDisk(config.PathDbWallet)
	if err != nil {
		logging.L.Warn().Err(err).
			Msg("startup failed, could setup full daemon")
	}
	d.Wallet = w

	return d, nil
}

func SetupNewInstancePrivateMode() (d *daemon.Daemon, err error) {
	setup := <-config.PrivateModeSetupChan
	d, err = daemon.SetupDaemonNoWallet()
	if err != nil {
		logging.L.Panic().Err(err).
			Msg("startup failed, could only produce daemon hull")
	}

	if setup.LabelCount == 0 {
		setup.LabelCount = 5
	}

	w, err := wallet.SetupWallet(setup.BirthHeight, setup.LabelCount, setup.ScanSecretKey, setup.SpendPubKey)
	if err != nil {
		logging.L.Err(err).
			Msg("startup failed, could not setup wallet")
		return nil, err
	}
	d.Wallet = w

	d.SetDbWriter(&database.DBWriter{Password: setup.Password})

	return d, nil
}
