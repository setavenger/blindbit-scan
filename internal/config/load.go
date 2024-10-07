package config

import (
	"encoding/hex"
	"log"
	"log/slog"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/goleveldb/leveldb/errors"
	"github.com/rs/zerolog"
	"github.com/setavenger/go-bip352"
	"github.com/spf13/viper"
)

func SetupConfigs(dirPath string) error {
	SetPaths(dirPath)
	return LoadConfigs(PathConfig)
}

func LoadConfigs(pathToConfig string) error {
	log.Println("loading configs from env and", pathToConfig)
	// Set the file name of the configurations file
	viper.SetConfigFile(pathToConfig)

	// Handle errors reading the config file
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Error reading config file, %s", err)
	}

	// map ENV var names
	viper.BindEnv("network.expose_http", "EXPOSE_HTTP")
	viper.BindEnv("network.blindbit_server", "BLINDBIT_SERVER")
	viper.BindEnv("network.electrum_server", "ELECTRUM_SERVER")
	viper.BindEnv("network.chain", "NETWORK_CHAIN")
	viper.BindEnv("network.electrum_tor", "ELECTRUM_TOR")
	viper.BindEnv("network.electrum_tor_proxy_host", "ELECTRUM_TOR_PROXY_HOST")

	viper.BindEnv("wallet.dust_limit", "WALLET_DUST_LIMIT")
	viper.BindEnv("wallet.label_count", "WALLET_LABEL_COUNT")
	viper.BindEnv("wallet.birth_height", "WALLET_BIRTH_HEIGHT")
	viper.BindEnv("wallet.scan_secret_key", "WALLET_SCAN_SECRET_KEY")
	viper.BindEnv("wallet.spend_pub_key", "WALLET_SPEND_PUB_KEY")

	viper.BindEnv("auth.user", "AUTH_USER")
	viper.BindEnv("auth.pass", "AUTH_PASS")

	viper.BindEnv("log_level", "LOG_LEVEL")

	// app seed is for umbrel inputs
	// viper.BindEnv("external_app_seed", "EXTERNAL_APP_SEED")

	/* set defaults */
	// network
	viper.SetDefault("network.expose_http", "127.0.0.1:8080")
	viper.SetDefault("network.blindbit_server", "http://localhost:8000")
	viper.SetDefault("network.electrum_server", "") // we set this to empty
	viper.SetDefault("network.chain", "signet")
	viper.SetDefault("network.electrum_tor", true)
	viper.SetDefault("network.electrum_tor_proxy_host", "127.0.0.1:9050")

	// wallet
	viper.SetDefault("wallet.dust_limit", 1000)
	viper.SetDefault("wallet.label_count", 1) // do at least the change label
	viper.SetDefault("wallet.birth_height", 840000)

	viper.SetDefault("log_level", "info")

	// app seed
	// viper.SetDefault("external_app_seed", "") // we normally don't use it

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

	// Basic Auth Data
	AuthUser = viper.GetString("auth.user")
	AuthPass = viper.GetString("auth.pass")

	if AuthUser == "" || AuthPass == "" {
		err := errors.New("config is missing auth settings")
		slog.Error(err.Error())
		return err
	}

	DustLimit = viper.GetUint64("wallet.dust_limit")

	var err error
	// load keys
	scanSecretStr := viper.GetString("wallet.scan_secret_key")
	if scanSecretStr != "" {
		var scanSecret []byte
		scanSecret, err = hex.DecodeString(scanSecretStr)
		if err != nil {
			log.Println(err)
			return err
		}
		ScanSecretKey = bip352.ConvertToFixedLength32(scanSecret)
	}

	spendPubStr := viper.GetString("wallet.spend_pub_key")
	if spendPubStr != "" {
		var spendPub []byte
		spendPub, err = hex.DecodeString(spendPubStr)
		if err != nil {
			log.Println(err)
			return err
		}
		SpendPubKey = bip352.ConvertToFixedLength33(spendPub)
	}

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
		log.Fatalf("Error parsing config, invalid chain: (%s)", chain)
	}

	logLevel := strings.ToLower(strings.TrimSpace(viper.GetString("log_level")))
	switch logLevel {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	return err
}
