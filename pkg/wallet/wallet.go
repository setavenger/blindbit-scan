package wallet

import (
	"encoding/json"
	"log"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/pkg/logging"
	"github.com/setavenger/blindbit-scan/pkg/types"
	"github.com/setavenger/blindbit-scan/pkg/utils"
	"github.com/setavenger/go-bip352"
)

type Wallet struct {
	SecretKeyScan  types.SecretKey `json:"sec_key_scan"`
	PubKeyScan     types.PublicKey `json:"pub_key_scan"`
	PubKeySpend    types.PublicKey `json:"pub_key_spend"`
	BirthHeight    uint64          `json:"birth_height,omitempty"`
	LastScanHeight uint64          `json:"last_scan,omitempty"`
	UTXOs          UtxoCollection  `json:"utxos,omitempty"`
	Labels         LabelMap        `json:"labels"`       // Labels contains all labels except for the change label
	UTXOMapping    UTXOMapping     `json:"utxo_mapping"` // used to keep track of utxos and not add the same twice
}

// This function is to create a new instance of a wallet.
// Reading a wallet from disk can simply be done via marshalling the stored data like any other struct.
func SetupWallet(
	birthHeight uint64,
	labelCount int,
	secretKeyScan [32]byte,
	pubKeySpend [33]byte,
) (wallet *Wallet, err error) {

	if birthHeight < 1 {
		birthHeight = 1
	}
	var lastScanHeight uint64
	if birthHeight-1 > 0 {
		// it needs to be 2 so that we can set last scan to 1 otherwise last scan has to be 1 anyways
		// last scan was last height which has already been processed. Scanning will continue with the lastScanHeight + 1
		lastScanHeight = birthHeight - 1
	} else {
		lastScanHeight = birthHeight
	}

	_, pubKeyScan := btcec.PrivKeyFromBytes(secretKeyScan[:])

	wallet = &Wallet{
		SecretKeyScan:  secretKeyScan,
		PubKeyScan:     bip352.ConvertToFixedLength33(pubKeyScan.SerializeCompressed()),
		PubKeySpend:    pubKeySpend,
		BirthHeight:    birthHeight,
		LastScanHeight: lastScanHeight,
		Labels:         LabelMap{},
		UTXOMapping:    UTXOMapping{},
	}

	// the user specifies the number of labels they have. So +1. Change label is m = 0
	for i := 0; i < labelCount+1; i++ {
		err = wallet.generateNextLabel()
		if err != nil {
			logging.L.Err(err).Msg("error generating labels")
			return nil, err
		}
	}
	return wallet, err
}

func (w *Wallet) Serialise() ([]byte, error) {
	return json.Marshal(w)
}

func (w *Wallet) DeSerialise(data []byte) error {
	return json.Unmarshal(data, w)
}

func (w *Wallet) AddUTXOs(utxos []*OwnedUTXO) error {
	for _, utxo := range utxos {
		key, err := utxo.GetKey()
		if err != nil {
			log.Println(err)
			return err
		}
		_, exists := w.UTXOMapping[key]
		if exists {
			continue
		}

		logging.L.Info().Hex("utxo", key[:]).Msg("new utxo added")

		w.UTXOs = append(w.UTXOs, utxo)
		w.UTXOMapping[key] = struct{}{}
	}

	return nil
}

func (w *Wallet) generateNextLabel() error {
	var mainnet bool
	if config.ChainParams.Name == chaincfg.MainNetParams.Name {
		mainnet = true
	}

	// we set the next m according to the length/ number of items in the labels map
	label, err := bip352.CreateLabel(w.SecretKeyScan, uint32(len(w.Labels)))
	if err != nil {
		return err
	}

	BmKey, err := bip352.AddPublicKeys(w.PubKeySpend, label.PubKey)
	if err != nil {
		logging.L.Err(err).Msg("")
		return err
	}
	address, err := bip352.CreateAddress(w.PubKeyScan, BmKey, mainnet, 0)
	if err != nil {
		return err
	}

	label.Address = address

	_, exists := w.Labels[label.PubKey]
	if exists {
		// users should not create the same label twice
		return utils.ErrLabelAlreadyExists
	}

	w.Labels[label.PubKey] = &label
	return err
}

func (w *Wallet) GetUTXOsByStates(states ...UTXOState) UtxoCollection {
	var utxos UtxoCollection
	for _, utxo := range w.UTXOs {
		for _, state := range states {
			if utxo.State == state {
				utxos = append(utxos, utxo)
			}
		}
	}
	return utxos
}

func (w *Wallet) FreeBalance() uint64 {
	var balance uint64 = 0
	for _, utxo := range w.UTXOs {
		if utxo.State == StateUnspent {
			balance += utxo.Amount
		}
	}
	return balance
}

func (w *Wallet) GetFreeUTXOs(includeSpentUnconfirmed bool) UtxoCollection {
	var utxos UtxoCollection
	for _, utxo := range w.UTXOs {
		if utxo.State == StateUnspent {
			utxos = append(utxos, utxo)
		}
		if includeSpentUnconfirmed && utxo.State == StateUnconfirmedSpent {
			utxos = append(utxos, utxo)
		}
	}
	return utxos
}

func (w *Wallet) GenerateAddress() (string, error) {
	var mainnet bool
	if config.ChainParams.Name == chaincfg.MainNetParams.Name {
		mainnet = true
	}
	address, err := bip352.CreateAddress(w.PubKeyScan, w.PubKeySpend, mainnet, 0)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return address, err
}
