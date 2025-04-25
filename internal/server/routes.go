package server

import (
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/pkg/database"
	"github.com/setavenger/blindbit-scan/pkg/logging"
	"github.com/setavenger/blindbit-scan/pkg/wallet"
	"github.com/setavenger/go-bip352"
	"github.com/spf13/viper"
)

func (s *Server) GetCurrentHeight(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"height": s.Daemon.Wallet.LastScanHeight})
}

func (s *Server) GetUtxos(c *gin.Context) {
	utxos := s.Daemon.Wallet.UTXOs
	if utxos == nil {
		utxos = []*wallet.OwnedUTXO{}
	}
	c.JSON(http.StatusOK, utxos)
}

func (s *Server) GetAddress(c *gin.Context) {
	address, err := s.Daemon.Wallet.GenerateAddress()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, gin.H{"address": address})
}

type RescanReq struct {
	Height uint64 `json:"height"`
}

func (s *Server) PostRescan(c *gin.Context) {
	var requestBody RescanReq
	err := c.ShouldBindJSON(&requestBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		c.Abort()
		return
	}

	if requestBody.Height < 1 {
		err = fmt.Errorf("height (%d) is invalid for rescan", requestBody.Height)
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		c.Abort()
		return
	}

	s.Daemon.TriggerRescanChan <- requestBody.Height
	c.JSON(http.StatusOK, gin.H{"height": requestBody.Height})
}

type SetupReq struct {
	ScanSecret  string `json:"scan_secret"`
	SpendPublic string `json:"spend_pub"`
	BirthHeight uint   `json:"birth_height"`
}

// todo: fix block after calling while sync is running
func (s *Server) PutSilentPaymentKeys(c *gin.Context) {
	var err error

	var keys SetupReq
	err = c.ShouldBindJSON(&keys)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		c.Abort()
		return
	}

	logging.L.Trace().Any("key", keys).Msg("")

	// load keys
	scanSecret, err := hex.DecodeString(keys.ScanSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		c.Abort()
		return
	}
	if len(scanSecret) != 32 {
		c.JSON(http.StatusInternalServerError, gin.H{"err": "scan secret must be 32 bytes"})
		c.Abort()
		return
	}

	spendPub, err := hex.DecodeString(keys.SpendPublic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		c.Abort()
		return
	}
	if len(spendPub) != 33 {
		c.JSON(http.StatusInternalServerError, gin.H{"err": "spend public key must be 33 bytes"})
		c.Abort()
		return
	}

	// we only write if nothing before failed
	if keys.BirthHeight < 1 {
		keys.BirthHeight = 1
	}

	viper.Set("wallet.scan_secret_key", keys.ScanSecret)
	viper.Set("wallet.spend_pub_key", keys.SpendPublic)
	viper.Set("wallet.birth_height", keys.BirthHeight)

	config.BirthHeight = uint64(keys.BirthHeight)
	config.ScanSecretKey = bip352.ConvertToFixedLength32(scanSecret)
	config.SpendPubKey = bip352.ConvertToFixedLength33(spendPub)

	// we write the changes to the config file.
	viper.SetConfigFile(config.PathConfig)
	err = viper.WriteConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		c.Abort()
		return
	}

	var newWallet *wallet.Wallet

	// logging.L.Trace().Any("birth", config.BirthHeight).Any("l-count", config.LabelCount).Any("scan", config.ScanSecretKey).Any("spend", config.SpendPubKey).Msg("config info")

	newWallet, err = wallet.SetupWallet(
		config.BirthHeight,
		config.LabelCount,
		config.ScanSecretKey,
		config.SpendPubKey,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		c.Abort()
		return
	}

	// reset system
	err = s.Daemon.ResetDaemonAndWallet()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		c.Abort()
		return
	}

	s.Daemon.Wallet = newWallet

	address, err := s.Daemon.Wallet.GenerateAddress()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		c.Abort()
		return
	}

	// no routine to alert the caller if something is wrong
	config.KeysReadyChan <- struct{}{}

	c.JSON(http.StatusOK, gin.H{"address": address})
}

func (s *Server) NewNwcConnection(c *gin.Context) {
	nwcURI, err := s.Nip47Controller.NewConnectionUri()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		c.Abort()
		return
	}

	err = s.Daemon.DBWriter.WriteNip47ControllerToDB(config.PathDbNWC, s.Nip47Controller)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{"uri": nwcURI})
}

// UnlockReq represents the request to unlock the wallet
type UnlockReq struct {
	Password string `json:"password"`
}

// UnlockResponse represents the response from unlocking the wallet
type UnlockResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (s *Server) Unlock(c *gin.Context) {
	var req UnlockReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// todo: several instances of password being passed around
	// todo: this should be refactored
	dbWriter := database.DBWriter{
		Password: req.Password,
	}

	s.Daemon.SetDbWriter(&dbWriter)

	// Decrypt wallet data
	if err := s.Daemon.Unlock(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlock wallet"})
		return
	}

	// Load auth credentials
	if err := s.Daemon.LoadAuthCredentials(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load auth credentials"})
		return
	}

	config.UnlockChan <- req.Password

	c.JSON(http.StatusOK, UnlockResponse{Success: true})
}

type SetupInstanceReq struct {
	ScanSecret  string `json:"scan_secret"`
	SpendPublic string `json:"spend_pub"`
	BirthHeight uint   `json:"birth_height"`
	Password    string `json:"password"`
}

// SetupInstance is used to setup the instance for the first time
// it will generate a new set of auth credentials and save them to the database
// the keys and the unlock password have to be sent in the request body
func (s *Server) SetupInstance(c *gin.Context) {
	var req SetupInstanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// load keys
	scanSecret, err := hex.DecodeString(req.ScanSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		c.Abort()
		return
	}
	if len(scanSecret) != 32 {
		c.JSON(http.StatusInternalServerError, gin.H{"err": "scan secret must be 32 bytes"})
		c.Abort()
		return
	}

	spendPub, err := hex.DecodeString(req.SpendPublic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		c.Abort()
		return
	}
	if len(spendPub) != 33 {
		c.JSON(http.StatusInternalServerError, gin.H{"err": "spend public key must be 33 bytes"})
		c.Abort()
		return
	}

	setup := config.PrivateModeSetup{
		ScanSecretKey: [32]byte(scanSecret),
		SpendPubKey:   [33]byte(spendPub),
		BirthHeight:   uint64(req.BirthHeight),
		Password:      req.Password,
	}

	config.PrivateModeSetupChan <- setup

	creds := config.GenerateAuthCredentials()
	config.SetAuthCredentials(creds)
	// Wait for auth credentials to be generated and loaded
	if creds == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate auth credentials"})
		return
	}

	tempWriter := database.DBWriter{
		Password: req.Password,
	}

	// Save auth credentials to disk
	if err := tempWriter.SaveAuthCredentials(creds); err != nil {
		logging.L.Error().Err(err).Msg("Failed to save auth credentials")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save auth credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"username": creds.Username,
		"password": creds.Password,
	})
}
