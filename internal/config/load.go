package config

import (
	"encoding/hex"
	"log"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/setavenger/go-bip352"
	"github.com/spf13/viper"
)

func SetupConfigs(dirPath string) {
	SetPaths(dirPath)
	LoadConfigs(PathConfig)
}

func LoadConfigs(pathToConfig string) error {
	// Set the file name of the configurations file
	viper.SetConfigFile(pathToConfig)

	// Handle errors reading the config file
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	/* set defaults */
	// network
	viper.SetDefault("network.expose_http", "127.0.0.1:8080")
	viper.SetDefault("network.blindbit_server", "http://localhost:8000")
	viper.SetDefault("network.electrum_server", "") // we set this to empty
	viper.SetDefault("network.chain", "signet")
	viper.SetDefault("network.electrum_tor", true)
	viper.SetDefault("network.electrum_tor_proxy_host", "127.0.0.1:9050")

	// wallet
	viper.SetDefault("wallet.minchange_amount", 1000)
	viper.SetDefault("wallet.dust_limit", 1000)
	viper.SetDefault("wallet.scan_only", false)
	viper.SetDefault("wallet.label_count", 1)      // do at least the change label
	viper.SetDefault("wallet.birthheight", 840000) // do at least the change label

	/* read and set config variables */
	ExposeHttpHost = viper.GetString("network.expose_http")
	BlindBitServerAddress = viper.GetString("network.blindbit_server")
	ElectrumServerAddress = viper.GetString("network.electrum_server")
	if ElectrumServerAddress != "" {
		UseElectrum = true
		useTor := viper.GetBool("network.electrum_tor")
		if useTor {
			ElectrumTorProxyHost = viper.GetString("network.electrum_tor_proxy_host")
		} else {
			// we set the host to empty which results in no tor being used
			ElectrumTorProxyHost = ""
		}
	} else {
		UseElectrum = false
		AutomaticScanInterval = 1 * time.Minute
	}

	DustLimit = viper.GetUint64("wallet.dust_limit")

	// load keys
	scanSecret, err := hex.DecodeString(viper.GetString("wallet.scan_secret_key"))
	if err != nil {
		log.Println(err)
		return err
	}
	ScanSecretKey = bip352.ConvertToFixedLength32(scanSecret)

	spendPub, err := hex.DecodeString(viper.GetString("wallet.spend_pub_key"))
	if err != nil {
		log.Println(err)
		return err
	}
	SpendPubKey = bip352.ConvertToFixedLength33(spendPub)

	LabelCount = viper.GetInt("wallet.label_count")
	BirthHeight = viper.GetUint64("wallet.birth_height")

	// extract the chain data and set the params
	chain := viper.GetString("network.chain")
	switch chain {
	case "main":
		ChainParams = &chaincfg.MainNetParams
	case "test":
		ChainParams = &chaincfg.TestNet3Params
	case "signet":
		ChainParams = &chaincfg.SigNetParams
	case "regtest":
		ChainParams = &chaincfg.RegressionNetParams
	default:
		log.Fatalf("Error reading config file, invalid chain: %s", chain)
	}

	return err
}
