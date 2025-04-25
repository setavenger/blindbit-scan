package startup

import (
	"github.com/setavenger/blindbit-scan/internal"
	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/internal/daemon"
	"github.com/setavenger/blindbit-scan/pkg/database"
	"github.com/setavenger/blindbit-scan/pkg/logging"
	"github.com/setavenger/blindbit-scan/pkg/wallet"
)

func StartupWithPrivateMode() (d *daemon.Daemon, err error) {
	if internal.CheckIfFileExists(config.PathDbWallet) {
		// we need to load the existing instance, wait for unlock api call
		d, err = SetupExistingInstancePrivateMode()
		if err != nil {
			logging.L.Panic().Err(err).
				Msg("startup failed, could produce daemon hull")
		}
	} else {
		// we need to setup a whole new instance and wait for Setup rest api call
		d, err = SetupNewInstancePrivateMode()
		if err != nil {
			logging.L.Panic().Err(err).
				Msg("startup failed, could produce daemon hull")
		}
	}

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

	d.Wallet = &wallet.Wallet{
		SecretKeyScan: setup.ScanSecretKey,
		PubKeySpend:   setup.SpendPubKey,
		BirthHeight:   setup.BirthHeight,
	}

	d.SetDbWriter(&database.DBWriter{Password: setup.Password})

	return d, nil
}
