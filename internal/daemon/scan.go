package daemon

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/btcsuite/btcd/btcutil/gcs"
	"github.com/btcsuite/btcd/btcutil/gcs/builder"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/pkg/database"
	"github.com/setavenger/blindbit-scan/pkg/logging"
	"github.com/setavenger/blindbit-scan/pkg/networking"
	"github.com/setavenger/blindbit-scan/pkg/utils" // todo move blindbitd/src to a pkg for all blindbit programs
	"github.com/setavenger/blindbit-scan/pkg/wallet"
	"github.com/setavenger/go-bip352"
)

type TweakScriptMap struct {
	Tweak        [33]byte
	ScriptPubKey [32]byte
}

func (d *Daemon) syncBlock(blockHeight uint64) ([]*wallet.OwnedUTXO, error) {
	tweaks, err := d.ClientBlindBit.GetTweaks(blockHeight, config.DustLimit)
	if err != nil {
		logging.L.Err(err).Msg("")
		return nil, err
	}

	labelsToCheck := make([]*bip352.Label, len(d.Wallet.Labels))
	i := 0
	for _, l := range d.Wallet.Labels {
		labelsToCheck[i] = l
		i++
	}

	// Map Tweaks to ScriptPubKey
	tweakToScriptMap := make(map[[32]byte]TweakScriptMap)

	for _, tweak := range tweaks {
		sharedSecret, err := bip352.CreateSharedSecret(tweak, d.Wallet.SecretKeyScan, nil)
		if err != nil {
			logging.L.Err(err).Msg("")
			return nil, err
		}

		outputPubKey, err := bip352.CreateOutputPubKey(sharedSecret, d.Wallet.PubKeySpend, 0)
		if err != nil {
			logging.L.Err(err).Msg("")
			return nil, err
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
				logging.L.Err(err).Msg("")
				panic(err)
			}

			tweakToScriptMap[bip352.ConvertToFixedLength32(labelPotentialOutputPrep[1:])] = TweakScriptMap{
				Tweak:        tweak,
				ScriptPubKey: bip352.ConvertToFixedLength32(labelPotentialOutputPrep[1:]),
			}

			negatedLabelPubKey, err := bip352.NegatePublicKey(label.PubKey)
			if err != nil {
				logging.L.Err(err).Msg("")
				panic(err)
			}

			labelPotentialOutputPrepNegated, err := bip352.AddPublicKeys(outputPubKey33, negatedLabelPubKey)
			if err != nil {
				logging.L.Err(err).Msg("")
				panic(err)
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

	// todo filter whether we should get the outputs in the first place

	// 	filterData, err := d.ClientBlindBit.GetFilter(blockHeight, networking.NewUTXOFilterType)
	// 	if err != nil {
	// 		logging.L.Err(err).Msg("")
	// 		return nil, err
	// 	}
	//
	// 	isMatch, err := matchFilter(filterData.Data, filterData.BlockHash, potentialOutputs)
	// 	if err != nil {
	// 		logging.L.Err(err).Msg("")
	// 		return nil, err
	// 	}
	// 	if !isMatch {
	// 		return nil, nil
	// 	}

	// Retrieve and Group Block Outputs by ScriptPubKey
	utxos, err := d.ClientBlindBit.GetUTXOs(blockHeight)
	if err != nil {
		logging.L.Err(err).Msg("")
		return nil, err
	}

	txidGroups := make(map[[32]byte][]*networking.UTXOServed) // txid -> utxos with that txid
	helperMapping := make(map[[32]byte][32]byte)              // helper mapping: output to txid
	for _, utxo := range utxos {
		txidGroups[utxo.Txid] = append(txidGroups[utxo.Txid], utxo)
		helperMapping[bip352.ConvertToFixedLength32(utxo.ScriptPubKey[2:])] = utxo.Txid
	}

	tweaksOutputsToCheckMap := make(map[[33]byte][]*networking.UTXOServed)
	// mapping from tweak -> txid (output group)
	for scriptPub32, tweakMap := range tweakToScriptMap {
		if txid, exists := helperMapping[scriptPub32]; exists {
			// scriptpubkey that we already precomputed is found within the utxos
			// we attach all outputs for the transaction to check
			if tweaksOutputsToCheckMap[tweakMap.Tweak], exists = txidGroups[txid]; !exists {
				err = fmt.Errorf("maps were not aligned, txid should always have utxos (%x)", txid[:])
				logging.L.Panic().Err(err).Msg("")
				return nil, err
			}
		}
	}

	// Scan Only Relevant Groups
	var ownedUTXOs []*wallet.OwnedUTXO

	for tweak, relevantUTXOs := range tweaksOutputsToCheckMap {
		var txOutputs [][32]byte
		for _, utxo := range relevantUTXOs {
			fixedLengthOutput := bip352.ConvertToFixedLength32(utxo.ScriptPubKey[2:])
			txOutputs = append(txOutputs, fixedLengthOutput)
		}

		foundOutputsPerTweak, err := bip352.ReceiverScanTransaction(
			d.Wallet.SecretKeyScan,
			d.Wallet.PubKeySpend,
			labelsToCheck,
			txOutputs,
			tweak,
			nil,
		)
		if err != nil {
			logging.L.Err(err).Msg("")
			return nil, err
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

	return ownedUTXOs, nil
}

func (d *Daemon) SyncToTip(chainTip uint64) error {
	var abort bool
	var err error
	if chainTip == 0 {
		chainTip, err = d.ClientBlindBit.GetChainTip()
		if err != nil {
			logging.L.Err(err).Msg("")
			return err
		}
	}

	logging.L.Debug().Msgf("Trying to sync to height: %d", chainTip)

	// todo find fixed points for mainnet/signet/testnet where startHeight can start from. Avoid scanning through non SP merged blocks
	var startHeight = d.Wallet.BirthHeight
	if d.Wallet.LastScanHeight >= startHeight {
		startHeight = d.Wallet.LastScanHeight + 1
	}

	if startHeight > chainTip {
		// todo debug/testing log
		return nil
	}

	// don't check genesis block
	if startHeight == 0 {
		startHeight = 1
	}

	go func() {
		<-d.ctx.Done()
		abort = true
	}()

	for i := startHeight; i < chainTip+1; i++ {
		if abort {
			err = d.ctx.Err()
			logging.L.Info().Msg("aborted sync")
			return err
		}

		err = d.MarkSpentUTXOs(i) // this can probably be omitted if electrum is used
		if err != nil {
			logging.L.Err(err).Uint64("height", i).Msg("error marking utxos")
			return err
		}
		// possible logging here to indicate to the user
		logging.L.Info().Uint64("height", i).Msg("syncing")
		var ownedUTXOs []*wallet.OwnedUTXO
		ownedUTXOs, err = d.syncBlock(i)
		if err != nil {
			logging.L.Err(err).Uint64("height", i).Msg("")
			return err
		}
		if ownedUTXOs == nil {
			if !abort {
				// otherwise the scan height will be overriden between key reset and cleaning up of sync loop
				// todo: make more robust, unnecessary trick
				d.Wallet.LastScanHeight = i
			}

			if i%100 == 0 {
				// do some writes anyways to save the last state of the scan height
				err = database.WriteWalletToDB(config.PathDbWallet, d.Wallet)
				if err != nil {
					logging.L.Err(err).Uint64("height", i).Msg("")
					return err
				}
			}
			continue
		}
		err = d.Wallet.AddUTXOs(ownedUTXOs)
		if err != nil {
			logging.L.Err(err).Msg("")
			return err
		}
		logging.L.Info().Msg("Added UTXOs to wallet")
		if !abort {
			// otherwise the scan height will be overriden between key reset and cleaning up of sync loop
			// todo: make more robust, unnecessary trick
			d.Wallet.LastScanHeight = i
		}

		// todo: database should be an interface to allow other forms of storing data.
		err = database.WriteWalletToDB(config.PathDbWallet, d.Wallet)
		if err != nil {
			logging.L.Err(err).Msg("")
			return err
		}
	}

	err = d.CheckUnspentUTXOs()
	if err != nil {
		logging.L.Err(err).Msg("")
		return err
	}
	return err
}

func (d *Daemon) ContinuousScan() (err error) {
	logging.L.Info().Msg("starting continous scan")

	ticker := time.NewTicker(config.AutomaticScanInterval)
	t1 := make(chan struct{}, 1)
	t1 <- struct{}{}

	var abort bool
	go func() {
		<-d.ctx.Done()
		abort = true
	}()

	for {
		if abort {
			err = d.ctx.Err()
			logging.L.Info().Msg("aborted continous scan")
			return err
		}

		select {
		case <-t1:
			// just for the initial trigger. Should only trigger once
			err := d.SyncToTip(0)
			if err != nil {
				logging.L.Err(err).Msg("could not sync to tip")
				// return err
			}
			logging.L.Info().Uint64("balance", d.Wallet.FreeBalance()).Msg("")
		case newBlock := <-d.NewBlockChan:
			<-time.After(5 * time.Second) // delay, indexing server does not index immediately after a block is found
			oldBalance := d.Wallet.FreeBalance()
			err := d.SyncToTip(uint64(newBlock.Height))
			if err != nil {
				logging.L.Err(err).Msg("could not sync to tip")
				// return err
			}
			newBalance := d.Wallet.FreeBalance()
			if oldBalance != newBalance {
				logging.L.Info().Uint64("balance", newBalance).Msg("update")
			}
		case height := <-d.TriggerRescanChan:
			oldBalance := d.Wallet.FreeBalance()
			err := d.ForceSyncFrom(height)
			if err != nil {
				logging.L.Err(err).Msg("could not sync to tip")
				return err
			}
			newBalance := d.Wallet.FreeBalance()
			if oldBalance != newBalance {
				logging.L.Info().Uint64("balance", newBalance).Msg("update")
			}
		case <-ticker.C:
			// todo is this needed if NewBlockChan is very robust?
			// check every 5 minutes anyway
			chainTip, err := d.ClientBlindBit.GetChainTip()
			if err != nil {
				logging.L.Err(err).Msg("could not get chain tip")
				// return err
			}

			if chainTip < d.Wallet.LastScanHeight {
				continue
			}

			err = d.SyncToTip(chainTip)
			if err != nil {
				logging.L.Err(err).Msg("could not get chain tip")
				// return err
			}
		case <-time.NewTicker(1 * time.Minute).C:
			if !config.UseElectrum {
				continue
			}
			// exclusively to check for spent UTXOs
			err := d.CheckUnspentUTXOs()
			if err != nil {
				logging.L.Err(err).Msg("could not check UTXO states")
				return err
			}
		}
	}
}

// CheckUnspentUTXOs
// checks against electrum whether unspent owned UTXOs are now unspent
func (d *Daemon) CheckUnspentUTXOs() error {
	if !config.UseElectrum {
		// we don't use Electrum if set to false
		logging.L.Warn().Msg("electrum is not configured")
		return nil
	}
	// todo this probably breaks if more than one UTXO are locked to a script
	//  this should never happen if the protocol is followed but still might occur
	for _, utxo := range d.Wallet.GetUTXOsByStates(wallet.StateUnspent, wallet.StateUnconfirmedSpent) {
		balance, err := d.ClientElectrum.GetBalance(context.Background(), utils.ConvertPubKeyToScriptHash(utxo.PubKey))
		if err != nil {
			logging.L.Err(err).Msg("")
			return err
		}
		if balance.Confirmed == 0.0 && balance.Unconfirmed == 0.0 {
			utxo.State = wallet.StateSpent
			continue
		}
		if balance.Unconfirmed < 0 {
			utxo.State = wallet.StateUnconfirmedSpent
			continue
		}
	}
	return nil
}

func (d *Daemon) MarkSpentUTXOs(blockHeight uint64) error {
	// move SpentOutpointsIndex to types
	filter, err := d.ClientBlindBit.GetFilter(blockHeight, networking.SpentOutpointsFilterType)
	if err != nil {
		logging.L.Err(err).Msg("")
		return err
	}
	hashes := d.generateLocalOutpointHashes([32]byte(filter.BlockHash))

	// convert to byte slice
	var hashesForFilter [][]byte
	for hash := range hashes {
		var newHash = make([]byte, 8)
		copy(newHash[:], hash[:])
		hashesForFilter = append(hashesForFilter, newHash[:])
	}

	isMatch, err := matchFilter(filter.Data, filter.BlockHash, hashesForFilter)
	if err != nil {
		logging.L.Err(err).Msg("")
		return err
	}

	if !isMatch {
		return nil
	}

	index, err := d.ClientBlindBit.GetSpentOutpointsIndex(blockHeight)
	if err != nil {
		logging.L.Err(err).Msg("")
		return err
	}

	for _, hash := range index.Data {
		if utxoPtr, ok := hashes[hash]; ok {
			utxoPtr.State = wallet.StateSpent
		}
	}

	return nil
}

func (d *Daemon) ForceSyncFrom(fromHeight uint64) error {
	chainTip, err := d.ClientBlindBit.GetChainTip()
	if err != nil {
		logging.L.Err(err).Msg("")
		return err
	}

	logging.L.Info().Msgf("ForceSyncFrom: %d to %d\n", fromHeight, chainTip)

	// don't check genesis block
	if fromHeight == 0 {
		fromHeight = 1
	}

	var abort bool
	go func() {
		<-d.ctx.Done()
		abort = true
	}()

	for i := fromHeight; i < chainTip+1; i++ {
		if abort {
			err = d.ctx.Err()
			logging.L.Info().Msg("aborted sync")
			return err
		}
		logging.L.Info().Uint64("height", i).Msg("syncing")
		err = d.MarkSpentUTXOs(i) // this can probably be omitted if electrum is used
		if err != nil {
			logging.L.Err(err).Uint64("height", i).Msg("error marking utxos")
			return err
		}

		// possible logging here to indicate to the user
		var ownedUTXOs []*wallet.OwnedUTXO
		ownedUTXOs, err = d.syncBlock(i)
		if err != nil {
			logging.L.Err(err).Msg("")
			return err
		}
		if ownedUTXOs == nil {
			if !abort {
				// otherwise the scan height will be overriden between key reset and cleaning up of sync loop
				// todo: make more robust, unnecessary trick
				d.Wallet.LastScanHeight = i
			}
			if i%100 == 0 {
				// do some writes anyways to save the last state of the scan height
				err = database.WriteWalletToDB(config.PathDbWallet, d.Wallet)
				if err != nil {
					logging.L.Err(err).Msg("")
					return err
				}
			}
			continue
		}
		err = d.Wallet.AddUTXOs(ownedUTXOs)
		if err != nil {
			logging.L.Err(err).Msg("")
			return err
		}
		if !abort {
			// otherwise the scan height will be overriden between key reset and cleaning up of sync loop
			// todo: make more robust, unnecessary trick
			d.Wallet.LastScanHeight = i
		}
		err = database.WriteWalletToDB(config.PathDbWallet, d.Wallet)
		if err != nil {
			logging.L.Err(err).Msg("")
			return err
		}
	}

	err = d.CheckUnspentUTXOs()
	if err != nil {
		logging.L.Err(err).Msg("")
		return err
	}
	log.Println("Rescan complete")
	log.Println("Balance:", d.Wallet.FreeBalance())
	return err
}

func (d *Daemon) generateLocalOutpointHashes(blockHash [32]byte) map[[8]byte]*wallet.OwnedUTXO {
	outputs := make(map[[8]byte]*wallet.OwnedUTXO, len(d.Wallet.UTXOs))
	blockHashLE := bip352.ReverseBytesCopy(blockHash[:])
	for _, utxo := range d.Wallet.UTXOs {
		if utxo.State == wallet.StateSpent {
			continue
		}
		var buf bytes.Buffer

		buf.Write(bip352.ReverseBytesCopy(utxo.Txid[:]))
		binary.Write(&buf, binary.LittleEndian, utxo.Vout)

		hashed := sha256.Sum256(append(buf.Bytes(), blockHashLE...))
		var shortHash [8]byte
		copy(shortHash[:], hashed[:])
		outputs[shortHash] = utxo
	}
	return outputs
}

func matchFilter(nBytes []byte, blockHash [32]byte, values [][]byte) (bool, error) {
	c := chainhash.Hash{}

	err := c.SetBytes(bip352.ReverseBytesCopy(blockHash[:]))
	if err != nil {
		logging.L.Err(err).Msg("")
		return false, err

	}

	filter, err := gcs.FromNBytes(builder.DefaultP, builder.DefaultM, nBytes)
	if err != nil {
		logging.L.Err(err).Msg("")
		return false, err
	}

	key := builder.DeriveKey(&c)

	isMatch, err := filter.HashMatchAny(key, values)
	if err != nil {
		logging.L.Err(err).Msg("")
		return false, err
	}

	if isMatch {
		return true, nil
	} else {
		return false, nil
	}
}
