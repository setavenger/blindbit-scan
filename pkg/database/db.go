package database

import (
	"os"

	"github.com/setavenger/blindbit-scan/internal/wallet"
	"github.com/setavenger/blindbit-scan/pkg/logging"
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
