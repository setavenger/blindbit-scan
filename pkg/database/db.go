package database

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/setavenger/blindbit-scan/internal"
	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/pkg/logging"
	"github.com/setavenger/blindbit-scan/pkg/networking/nwc"
	"github.com/setavenger/blindbit-scan/pkg/types"
	"github.com/setavenger/blindbit-scan/pkg/wallet"
)

type Serialiser interface {
	Serialise() ([]byte, error)
	DeSerialise([]byte) error
}

type DBWriter struct {
	Password string
}

func (w *DBWriter) WriteToDB(path string, dataStruct Serialiser) error {
	if config.PrivateMode {
		data, err := dataStruct.Serialise()
		if err != nil {
			logging.L.Err(err).Msg("failed to serialise data")
			return err
		}
		data, err = EncryptData(data, w.Password)
		if err != nil {
			logging.L.Err(err).Msg("failed to encrypt data")
			return err
		}
		err = os.WriteFile(path, data, 0644)
		if err != nil {
			logging.L.Err(err).Msg("failed to write to db")
			return err
		}
		return nil
	}
	return WriteToDB(path, dataStruct)
}

func (w *DBWriter) ReadFromDB(path string, dataStruct Serialiser) error {
	if config.PrivateMode {
		data, err := os.ReadFile(path)
		if err != nil {
			logging.L.Err(err).Msg("failed to read from db")
			return err
		}
		data, err = DecryptData(data, w.Password)
		if err != nil {
			logging.L.Err(err).Msg("failed to decrypt data")
			return err
		}
		return dataStruct.DeSerialise(data)
	}
	return ReadFromDB(path, dataStruct)
}

func (w *DBWriter) EncryptData(data []byte) ([]byte, error) {
	return EncryptData(data, w.Password)
}

// AES GCM encryption function
func EncryptData(data []byte, password string) ([]byte, error) {
	// Hash the password to get a 32-byte key
	key := sha256.Sum256([]byte(password))

	// Create a new AES cipher block using the hashed key
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	// Create a new GCM cipher mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Generate a random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt the data using GCM
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func DecryptData(data []byte, password string) ([]byte, error) {
	// Hash the password to get a 32-byte key
	key := sha256.Sum256([]byte(password))

	// Create a new AES cipher block using the hashed key
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	// Create a new GCM cipher mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Get the nonce size
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce from ciphertext
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	// Decrypt the data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
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

func (w *DBWriter) WriteWalletToDB(p string, wallet *wallet.Wallet) error {
	if wallet == nil {
		// do nothing
		logging.L.Warn().Msg("wallet was nil")
		return nil
	}
	return w.WriteToDB(p, wallet)
}

func (w *DBWriter) WriteNip47ControllerToDB(p string, c *nwc.Nip47Controller) error {
	return w.WriteToDB(p, c)
}

func (w *DBWriter) LoadWalletFromDisk(path string) (*wallet.Wallet, error) {
	if internal.CheckIfFileExists(path) {
		var wallet wallet.Wallet
		err := w.ReadFromDB(path, &wallet)
		return &wallet, err
	}

	logging.L.Trace().Str("path", path).Msg("No wallet data on disk")

	return nil, errors.New("no wallet data on disk")
}

func (w *DBWriter) TryLoadingControllerFromDisk(
	ctx context.Context,
	path string,
) (
	c *nwc.Nip47Controller,
	err error,
) {
	if internal.CheckIfFileExists(path) {
		var apps nwc.Apps
		err = w.ReadFromDB(path, &apps)
		if err != nil {
			logging.L.Err(err).Msg("failed to load apps from db")
			return nil, err
		}

		return nwc.NewNip47ControllerFromApps(ctx, apps), nil
	}

	logging.L.Trace().Str("path", path).Msg("No NWC apps data on disk")

	return nwc.NewNip47Controller(ctx), nil
}

func (w *DBWriter) SaveAuthCredentials(creds *types.AuthCredentials) error {
	return w.WriteToDB(config.PathDbAuth, creds)
}

func (w *DBWriter) LoadAuthCredentials() (*types.AuthCredentials, error) {
	var creds types.AuthCredentials
	err := w.ReadFromDB(config.PathDbAuth, &creds)
	return &creds, err
}
