package startup

import (
	"github.com/setavenger/blindbit-scan/internal"
	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/internal/daemon"
	"github.com/setavenger/blindbit-scan/pkg/database"
	"github.com/setavenger/blindbit-scan/pkg/logging"
	"github.com/setavenger/blindbit-scan/pkg/wallet"
)

func StartupWithSimpleMode() (d *daemon.Daemon, err error) {
	if internal.CheckIfFileExists(config.PathDbWallet) {
		d, err = SetupExistingInstanceSimpleMode()
	} else {
		d, err = SetupNewInstanceSimpleMode()
	}
	if err != nil {
		logging.L.Panic().Err(err).
			Msg("startup failed, could only produce daemon hull")
	}

	return d, nil
}

func SetupNewInstanceSimpleMode() (d *daemon.Daemon, err error) {
	// no password needed, so we just do the old process
	d, err = daemon.SetupDaemonNoWallet()
	if err != nil {
		logging.L.Panic().Err(err).
			Msg("startup failed, could only produce daemon hull")
	}

	// no password needed, so we just set it to empty string
	d.SetDbWriter(&database.DBWriter{Password: ""})

	if config.ScanSecretKey != [32]byte{} && config.SpendPubKey != [33]byte{} {
		w, err := wallet.SetupWallet(config.BirthHeight, config.LabelCount, config.ScanSecretKey, config.SpendPubKey)
		if err != nil {
			logging.L.Panic().Err(err).
				Msg("startup failed, could not setup wallet")
		}
		d.Wallet = w
	}
	return d, nil
}

func SetupExistingInstanceSimpleMode() (d *daemon.Daemon, err error) {
	// no password needed, so we just do the old process
	d, err = daemon.SetupDaemonNoWallet()
	if err != nil {
		logging.L.Panic().Err(err).
			Msg("startup failed, could only produce daemon hull")
	}

	// no password needed, so we just set it to empty string
	d.SetDbWriter(&database.DBWriter{Password: ""})

	w, err := d.DBWriter.LoadWalletFromDisk(config.PathDbWallet)
	if err != nil {
		logging.L.Warn().Err(err).
			Msg("startup failed, could setup full daemon")
	}
	d.Wallet = w

	return d, nil
}
