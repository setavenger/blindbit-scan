package scan

import (
	"bytes"

	"github.com/setavenger/blindbit-scan/pkg/networking"
	"github.com/setavenger/blindbit-scan/pkg/wallet"
	"github.com/setavenger/go-bip352"
)

type Scanner interface {
	SpendPubKey() [33]byte
	ScanSecretKey() [32]byte
	Labels() []*bip352.Label
}

// IdentifyTxOuputs only returns the output pubkey
// Those need to be matched with original utxo data for meta data
func IdentifyTxOuputs(
	s Scanner,
	outputs [][32]byte,
	tweaks [][33]byte,
) (
	foundOutputs []*bip352.FoundOutput,
	err error,
) {
	for _, tweak := range tweaks {
		foundOutputsPerTweak, err := bip352.ReceiverScanTransaction(
			s.ScanSecretKey(),
			s.SpendPubKey(),
			s.Labels(),
			outputs,
			tweak,
			nil,
		)
		if err != nil {
			return nil, err
		}

		foundOutputsPerTweak = append(foundOutputsPerTweak, foundOutputsPerTweak...)
	}

	return
}

func ScanData(
	s Scanner,
	utxos []*networking.UTXOServed,
	tweaks [][33]byte,
) (
	ownedUTXOs []*wallet.OwnedUTXO,
	err error,
) {
	outputs := make([][32]byte, len(utxos))

	for i := range utxos {
		copy(outputs[i][:], utxos[i].ScriptPubKey[2:])
	}

	var foundOutputs []*bip352.FoundOutput
	foundOutputs, err = IdentifyTxOuputs(s, outputs, tweaks)
	if err != nil {
		return nil, err
	}

	if len(foundOutputs) == 0 {
		// nothing found so we exit no issue
		return
	}

	for _, foundOutput := range foundOutputs {
		for _, utxo := range utxos {
			if bytes.Equal(foundOutput.Output[:], utxo.ScriptPubKey[2:]) {
				state := wallet.StateUnspent
				if utxo.Spent {
					state = wallet.StateSpent
				}
				ownedUTXOs = append(ownedUTXOs, &wallet.OwnedUTXO{
					Txid:         utxo.Txid,
					Vout:         utxo.Vout,
					Amount:       utxo.Amount,
					PrivKeyTweak: foundOutput.SecKeyTweak,
					PubKey:       foundOutput.Output,
					Timestamp:    utxo.Timestamp,
					State:        state,
					Label:        foundOutput.Label,
				})
				break
			}
		}
	}

	return
}
