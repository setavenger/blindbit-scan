package server

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/internal/wallet"
	"github.com/setavenger/blindbit-scan/pkg/logging"
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
	ScanSecret  string `json:"secret_sec"`
	SpendPublic string `json:"spend_pub"`
	BirthHeight uint   `json:"birth_height"`
}

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
	viper.Set("wallet.scan_secret_key", keys.ScanSecret)

	spendPub, err := hex.DecodeString(keys.SpendPublic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		c.Abort()
		return
	}
	viper.Set("wallet.spend_pub_key", keys.SpendPublic)
	viper.Set("wallet.birth_height", keys.BirthHeight)

	// we only write if nothing before failed
	if keys.BirthHeight < 1 {
		keys.BirthHeight = 1
	}
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

	go func() {
		if s.Daemon.Wallet == nil || bytes.Equal(s.Daemon.Wallet.SecretKeyScan[:], make([]byte, 32)) || bytes.Equal(s.Daemon.Wallet.PubKeySpend[:], make([]byte, 33)) {
			config.KeysReadyChan <- struct{}{}
		}
	}()

	var newWallet *wallet.Wallet

	// logging.L.Trace().Any("birth", config.BirthHeight).Any("l-count", config.LabelCount).Any("scan", config.ScanSecretKey).Any("spend", config.SpendPubKey).Msg("config info")

	newWallet, err = wallet.SetupWallet(config.BirthHeight, config.LabelCount, config.ScanSecretKey, config.SpendPubKey)
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

	// logging.L.Debug().Any("wallet", s.Daemon.Wallet).Msg("")

	go func() {
		<-time.After(5 * time.Second)
		err = s.Daemon.ContinuousScan()
		if err != nil {
			logging.L.Err(err).Msg("")
			return
		}
	}()

	// logging.L.Trace().Any("wallet", s.Daemon.Wallet).Msg("")

	address, err := s.Daemon.Wallet.GenerateAddress()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, gin.H{"address": address})
}
