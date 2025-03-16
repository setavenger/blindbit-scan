package database

import (
	"context"
	"os"

	"github.com/setavenger/blindbit-scan/internal"
	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/pkg/logging"
	"github.com/setavenger/blindbit-scan/pkg/networking/nwc"
	"github.com/setavenger/blindbit-scan/pkg/wallet"
)

type Serialiser interface {
	Serialise() ([]byte, error)
	DeSerialise([]byte) error
}

func WriteToDB(path string, dataStruct Serialiser) error {
	data, err := dataStruct.Serialise()
	if err != nil {
		logging.L.Err(err).Msg("")
		return err
	}
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		logging.L.Err(err).Msg("")
		return err
	}

	return nil
}

func ReadFromDB(path string, dataStruct Serialiser) error {
	data, err := os.ReadFile(path)
	if err != nil {
		logging.L.Err(err).Msg("")
		return err
	}

	err = dataStruct.DeSerialise(data)
	if err != nil {
		logging.L.Err(err).Msg("")
		return err
	}

	return nil
}

func WriteWalletToDB(p string, w *wallet.Wallet) error {
	if w == nil {
		// do nothting
		logging.L.Warn().Msg("wallet was nil")
		return nil
	}
	return WriteToDB(p, w)
}

func WriteNip47ControllerToDB(p string, c *nwc.Nip47Controller) error {
	return WriteToDB(p, c)
}

func TryLoadWalletFromDisk(path string) (*wallet.Wallet, error) {
	if internal.CheckIfFileExists(path) {
		var w wallet.Wallet
		err := ReadFromDB(path, &w)
		return &w, err
	}

	logging.L.Trace().Str("path", path).Msg("No wallet data on disk")

	return wallet.SetupWallet(
		config.BirthHeight,
		config.LabelCount,
		config.ScanSecretKey,
		config.SpendPubKey,
	)
}

func TryLoadingControllerFromDisk(
	ctx context.Context,
	path string,
) (
	c *nwc.Nip47Controller,
	err error,
) {
	if internal.CheckIfFileExists(path) {
		var apps nwc.Apps
		err = ReadFromDB(path, &apps)
		if err != nil {
			logging.L.Err(err).Msg("failed to load apps from db")
			return nil, err
		}

		logging.L.Trace().Any("apps", apps).Msg("")

		return nwc.NewNip47ControllerFromApps(ctx, apps), nil
	}

	logging.L.Trace().Str("path", path).Msg("No NWC apps data on disk")

	return nwc.NewNip47Controller(ctx), nil
}
