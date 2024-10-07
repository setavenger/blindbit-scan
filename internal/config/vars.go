package config

import (
	"time"

	"github.com/btcsuite/btcd/chaincfg"
)

func init() {
	KeysReadyChan = make(chan struct{})
}

var (
	// ExposeHttpHost if set gRPC will be exposed via http and not unix socket. This variable also defines the where it will be exposed.
	ExposeHttpHost string

	// BlindBitServerAddress Indexing server for silent payments that follows the blindbit standard
	BlindBitServerAddress string

	// ElectrumServerAddress Electrum server
	ElectrumServerAddress string

	// DustLimit only receives amounts and checks tweaks where the biggest utxo exceeds the dust limit.
	// Note: that if you receive funds below this threshold you might not find them.
	// Rescan with DustLimit = 0 to find those.
	DustLimit uint64

	// ChainParams defines on which chain the wallet runs
	ChainParams *chaincfg.Params

	// ElectrumTorProxyHost if the host addr is given, tor will be used normally "127.0.0.1:9050". This is also the default setting
	ElectrumTorProxyHost = ""

	// UseElectrum no electrum calls will be made if false. Setting an electrum address wil set to true in settings.
	UseElectrum bool

	// AutomaticScanInterval has different values depending on whether Electrum is used or not
	AutomaticScanInterval time.Duration = 5 * time.Minute // 5 minutes if electrum is active

	ScanSecretKey [32]byte

	SpendPubKey [33]byte

	BirthHeight uint64

	LabelCount int

	// basic auth details
	AuthUser string

	AuthPass string

	// keys are ready chan
	KeysReadyChan chan struct{}
)
