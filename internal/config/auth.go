// internal/config/auth.go
package config

import (
	"crypto/rand"

	"github.com/btcsuite/btcd/btcutil/base58"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/setavenger/blindbit-scan/pkg/logging"
	"github.com/setavenger/blindbit-scan/pkg/types"
)

var (
	authCredentials *types.AuthCredentials
)

func GenerateAuthCredentials() *types.AuthCredentials {
	username := petname.Generate(2, "-")
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		logging.L.Panic().Err(err).Msg("failed to generate random bytes")
		return nil
	}
	password := base58.Encode(randomBytes)

	return &types.AuthCredentials{
		Username: username,
		Password: password,
	}
}

func SetAuthCredentials(creds *types.AuthCredentials) {
	authCredentials = creds
}

func GetAuthCredentials() *types.AuthCredentials {
	return authCredentials
}
