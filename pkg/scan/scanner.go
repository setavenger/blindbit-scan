package scan

import (
	"bytes"
	"fmt"
	"time"

	"github.com/setavenger/blindbit-scan/pkg/networking"
	"github.com/setavenger/blindbit-scan/pkg/wallet"
	"github.com/setavenger/go-bip352"
)

type Scanner interface {
	SpendPubKey() [33]byte
	ScanSecretKey() [32]byte
	Labels() []*bip352.Label
}

type TweakScriptMap struct {
	Tweak        [33]byte
	ScriptPubKey [32]byte
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

		foundOutputs = append(foundOutputs, foundOutputsPerTweak...)
	}

	return
}

// ScanDataOptimized is an optimized version that precomputes script pubkeys
// and only scans relevant UTXOs, similar to the syncBlock function
func ScanDataOptimized(
	s Scanner,
	utxos []*networking.UTXOServed,
	tweaks [][33]byte,
) (
	ownedUTXOs []*wallet.OwnedUTXO,
	err error,
) {
	if len(tweaks) == 0 {
		return nil, nil
	}

	labelsToCheck := s.Labels()

	// Time the entire operation
	start := time.Now()

	// Map Tweaks to ScriptPubKey - precompute all possible script pubkeys
	tweakToScriptMap := make(map[[32]byte]TweakScriptMap)

	for _, tweak := range tweaks {
		sharedSecret, err := bip352.CreateSharedSecret(tweak, s.ScanSecretKey(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create shared secret: %w", err)
		}

		outputPubKey, err := bip352.CreateOutputPubKey(sharedSecret, s.SpendPubKey(), 0)
		if err != nil {
			return nil, fmt.Errorf("failed to create output pubkey: %w", err)
		}

		tweakToScriptMap[outputPubKey] = TweakScriptMap{
			Tweak:        tweak,
			ScriptPubKey: outputPubKey,
		}

		// also precompute for labels
		for _, label := range labelsToCheck {
			outputPubKey33 := bip352.ConvertToFixedLength33(append([]byte{0x02}, outputPubKey[:]...))
			labelPotentialOutputPrep, err := bip352.AddPublicKeys(outputPubKey33, label.PubKey)
			if err != nil {
				return nil, fmt.Errorf("failed to add public keys: %w", err)
			}

			tweakToScriptMap[bip352.ConvertToFixedLength32(labelPotentialOutputPrep[1:])] = TweakScriptMap{
				Tweak:        tweak,
				ScriptPubKey: bip352.ConvertToFixedLength32(labelPotentialOutputPrep[1:]),
			}

			negatedLabelPubKey, err := bip352.NegatePublicKey(label.PubKey)
			if err != nil {
				return nil, fmt.Errorf("failed to negate public key: %w", err)
			}

			labelPotentialOutputPrepNegated, err := bip352.AddPublicKeys(outputPubKey33, negatedLabelPubKey)
			if err != nil {
				return nil, fmt.Errorf("failed to add negated public keys: %w", err)
			}

			tweakToScriptMap[bip352.ConvertToFixedLength32(labelPotentialOutputPrepNegated[1:])] = TweakScriptMap{
				Tweak:        tweak,
				ScriptPubKey: bip352.ConvertToFixedLength32(labelPotentialOutputPrepNegated[1:]),
			}
		}
	}

	if len(tweakToScriptMap) == 0 {
		return nil, nil
	}

	// Group UTXOs by transaction ID for efficient scanning
	txidGroups := make(map[[32]byte][]*networking.UTXOServed) // txid -> utxos with that txid
	helperMapping := make(map[[32]byte][32]byte)              // helper mapping: output to txid
	for _, utxo := range utxos {
		txidGroups[utxo.Txid] = append(txidGroups[utxo.Txid], utxo)
		helperMapping[bip352.ConvertToFixedLength32(utxo.ScriptPubKey[2:])] = utxo.Txid
	}

	// Map tweaks to relevant UTXOs to check
	tweaksOutputsToCheckMap := make(map[[33]byte][]*networking.UTXOServed)
	// mapping from tweak -> txid (output group)
	for scriptPub32, tweakMap := range tweakToScriptMap {
		if txid, exists := helperMapping[scriptPub32]; exists {
			// scriptpubkey that we already precomputed is found within the utxos
			// we attach all outputs for the transaction to check
			if tweaksOutputsToCheckMap[tweakMap.Tweak], exists = txidGroups[txid]; !exists {
				return nil, fmt.Errorf("maps were not aligned, txid should always have utxos (%x)", txid[:])
			}
		}
	}

	// Scan Only Relevant Groups
	for tweak, relevantUTXOs := range tweaksOutputsToCheckMap {
		var txOutputs [][32]byte
		for _, utxo := range relevantUTXOs {
			fixedLengthOutput := bip352.ConvertToFixedLength32(utxo.ScriptPubKey[2:])
			txOutputs = append(txOutputs, fixedLengthOutput)
		}

		foundOutputsPerTweak, err := bip352.ReceiverScanTransaction(
			s.ScanSecretKey(),
			s.SpendPubKey(),
			labelsToCheck,
			txOutputs,
			tweak,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		for _, foundOutput := range foundOutputsPerTweak {
			for _, utxo := range relevantUTXOs {
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
	}

	// Log timing and stats
	duration := time.Since(start)
	totalOps := len(tweaks) * (1 + len(labelsToCheck)*2) // base + (normal + negated) labels
	fmt.Printf("ScanDataOptimized: %v for %d tweaks, %d labels, %d total ops, %d utxos, %d found\n",
		duration, len(tweaks), len(labelsToCheck), totalOps, len(utxos), len(ownedUTXOs))

	return ownedUTXOs, nil
}

// ScanData is the original implementation for backward compatibility
// Deprecated: Use ScanDataOptimized instead
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
