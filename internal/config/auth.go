// internal/config/auth.go
package config

import (
	"crypto/rand"

	"github.com/btcsuite/btcd/btcutil/base58"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/setavenger/blindbit-scan/pkg/types"
)

var (
	authCredentials *types.AuthCredentials
)

func GenerateAuthCredentials() *types.AuthCredentials {
	username := petname.Generate(2, "-")
	randomBytes := make([]byte, 16)

	rand.Read(randomBytes) // never returns an error

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
