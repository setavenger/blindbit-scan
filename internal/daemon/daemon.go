package daemon

import (
	"context"
	"errors"
	"os"

	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/pkg/database"
	"github.com/setavenger/blindbit-scan/pkg/logging"
	"github.com/setavenger/blindbit-scan/pkg/networking" // todo move all blindbitd/src/*
	"github.com/setavenger/blindbit-scan/pkg/wallet"
	"github.com/setavenger/go-electrum/electrum"
)

type Daemon struct {
	ctx               context.Context
	cancelFunc        context.CancelFunc
	ShutdownChan      chan struct{}
	ClientElectrum    *electrum.Client
	ClientBlindBit    *networking.ClientBlindBit
	Wallet            *wallet.Wallet
	NewBlockChan      <-chan *electrum.SubscribeHeadersResult
	TriggerRescanChan chan uint64
}

// Will try to load a wallet from disk or will create a new one based on the blindbit.toml config-file
func SetupDaemon(path string) (*Daemon, error) {
	clientBlindBit := networking.ClientBlindBit{BaseUrl: config.BlindBitServerAddress}
	var clientElectrum *electrum.Client
	var err error

	if config.UseElectrum {
		logging.L.Info().Msg("connecting to Electrum server")
		clientElectrum, err = networking.CreateElectrumClient(config.ElectrumServerAddress, config.ElectrumTorProxyHost)
		if err != nil {
			logging.L.Err(err).Msg("")
			return nil, err
		}
	}

	w, err := database.TryLoadWalletFromDisk(path)
	if err != nil {
		logging.L.Err(err).Msg("")
		return nil, err
	}
	d, err := NewDaemon(w, &clientBlindBit, clientElectrum)
	if err != nil {
		logging.L.Err(err).Msg("")
		return nil, err
	}

	return d, err
}

func NewDaemon(wallet *wallet.Wallet, clientBlindBit *networking.ClientBlindBit, clientElectrum *electrum.Client) (*Daemon, error) {
	var channel <-chan *electrum.SubscribeHeadersResult
	var err error
	if config.UseElectrum {
		channel, err = clientElectrum.SubscribeHeaders(context.Background())
		if err != nil {
			logging.L.Err(err).Msg("")
			return nil, err
		}
	}

	daemon := Daemon{
		Wallet:            wallet,
		ClientBlindBit:    clientBlindBit,
		ClientElectrum:    clientElectrum,
		ShutdownChan:      make(chan struct{}),
		NewBlockChan:      channel,
		TriggerRescanChan: make(chan uint64),
	}
	ctx, cancel := context.WithCancel(context.Background())
	daemon.ctx = ctx
	daemon.cancelFunc = cancel

	return &daemon, nil
}

// SetupDaemonNoWallet is more of a helper should not be used
//
//	 todo: remove and change the flow of the program.
//		have proper handling of non existent keys on the first startup
func SetupDaemonNoWallet() (*Daemon, error) {
	clientBlindBit := networking.ClientBlindBit{BaseUrl: config.BlindBitServerAddress}
	var clientElectrum *electrum.Client
	var err error

	if config.UseElectrum {
		logging.L.Info().Msg("connecting to Electrum server")
		clientElectrum, err = networking.CreateElectrumClient(config.ElectrumServerAddress, config.ElectrumTorProxyHost)
		if err != nil {
			logging.L.Err(err).Msg("")
			return nil, err
		}
	}

	d, err := NewDaemon(nil, &clientBlindBit, clientElectrum)
	if err != nil {
		logging.L.Err(err).Msg("")
		return nil, err
	}

	return d, err
}

// ResetDaemonAndWallet deletes the stored wallet DB
// used when new keys are added such that scanning continues from scratch
func (d *Daemon) ResetDaemonAndWallet() (err error) {
	d.Cancel()
	err = os.Remove(config.PathDbWallet)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		logging.L.Err(err).Msg("")
		return
	}
	// clean out the error
	err = nil
	return
}

func (d *Daemon) Cancel() {
	d.cancelFunc()
	ctx, cancel := context.WithCancel(context.Background())
	d.ctx = ctx
	d.cancelFunc = cancel
}

func (d *Daemon) SaveWalletToDB() (err error) {
	return database.WriteWalletToDB(config.PathDbWallet, d.Wallet)
}
