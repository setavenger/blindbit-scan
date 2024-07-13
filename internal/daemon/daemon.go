package daemon

import (
	"context"
	"log"

	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/internal/wallet"
	"github.com/setavenger/blindbit-scan/pkg/networking" // todo move all blindbitd/src/*
	"github.com/setavenger/go-electrum/electrum"
)

type Daemon struct {
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
		log.Println("connecting to Electrum server")
		clientElectrum, err = networking.CreateElectrumClient(config.ElectrumServerAddress, config.ElectrumTorProxyHost)
		if err != nil {
			log.Println(err)
			return nil, err
		}
	}

	w, err := wallet.TryLoadWalletFromDisk(path)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	d, err := NewDaemon(w, &clientBlindBit, clientElectrum)
	if err != nil {
		log.Println(err)
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
			log.Println(err)
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
	return &daemon, nil
}
