package daemon

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"log"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil/gcs"
	"github.com/btcsuite/btcutil/gcs/builder"
	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/internal/wallet"
	"github.com/setavenger/blindbit-scan/pkg/database"
	"github.com/setavenger/blindbit-scan/pkg/networking"
	"github.com/setavenger/blindbit-scan/pkg/utils"
	"github.com/setavenger/blindbitd/src" // todo move blindbitd/src to a pkg for all blindbit programs
	"github.com/setavenger/go-bip352"
)

func (d *Daemon) syncBlock(blockHeight uint64) ([]*wallet.OwnedUTXO, error) {

	tweaks, err := d.ClientBlindBit.GetTweaks(blockHeight, config.DustLimit)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	labelsToCheck := make([]*bip352.Label, len(d.Wallet.Labels))
	i := 0
	for _, l := range d.Wallet.Labels {
		labelsToCheck[i] = l
		i++
	}

	// todo change back to assigning via index slice[i] once we are sure how long a slice will be; can we be sure how long it will always be?
	var potentialOutputs [][]byte
	// check for all tweaks normal outputs
	// + all tweaks * labels
	// + all tweaks * labels with opposite parity
	// + all prior found scripts in case someone resent to one of those scripts
	//var potentialOutputs = make([][]byte, len(tweaks)*(len(labelsToCheck)*2)+len(d.Wallet.PubKeysToWatch))

	for _, tweak := range tweaks {
		var sharedSecret [33]byte
		sharedSecret, err = bip352.CreateSharedSecret(tweak, d.Wallet.SecretKeyScan(), nil)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		var outputPubKey [32]byte
		outputPubKey, err = bip352.CreateOutputPubKey(sharedSecret, d.Wallet.PubKeySpend, 0)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		// todo we do this for now until the filters are changed to the 32byte x-only taproot pub keys (resolved)
		potentialOutputs = append(potentialOutputs, outputPubKey[:])
		// todo add option to skip precomputed labels and just pull all found outputs directly.
		//  For large numbers of labels and situations where bandwidth is not a constrained
		for _, label := range labelsToCheck {

			outputPubKey33 := bip352.ConvertToFixedLength33(append([]byte{0x02}, outputPubKey[:]...))

			// even parity
			var labelPotentialOutputPrep [33]byte
			labelPotentialOutputPrep, err = bip352.AddPublicKeys(outputPubKey33, label.PubKey)
			if err != nil {
				log.Println(err)
				panic(err)
			}

			potentialOutputs = append(potentialOutputs, labelPotentialOutputPrep[1:])

			// add label with uneven parity as well
			var negatedLabelPubKey [33]byte
			negatedLabelPubKey, err = bip352.NegatePublicKey(label.PubKey)
			if err != nil {
				log.Println(err)
				panic(err)
			}

			var labelPotentialOutputPrepNegated [33]byte
			labelPotentialOutputPrepNegated, err = bip352.AddPublicKeys(outputPubKey33, negatedLabelPubKey)
			if err != nil {
				log.Println(err)
				panic(err)
			}

			potentialOutputs = append(potentialOutputs, labelPotentialOutputPrepNegated[1:])
		}
	}

	if len(potentialOutputs) == 0 {
		return nil, nil
	}

	filterData, err := d.ClientBlindBit.GetFilter(blockHeight, networking.NewUTXOFilterType)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	isMatch, err := matchFilter(filterData.Data, filterData.BlockHash, potentialOutputs)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if !isMatch {
		return nil, nil
	}

	utxos, err := d.ClientBlindBit.GetUTXOs(blockHeight)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var foundOutputs []*bip352.FoundOutput

	var blockOutputs = make([][32]byte, len(utxos)) // we use it as txOutputs we check against all outputs from the block
	for i, utxo := range utxos {
		blockOutputs[i] = bip352.ConvertToFixedLength32(utxo.ScriptPubKey[2:])
	}

	for _, tweak := range tweaks {
		var foundOutputsPerTweak []*bip352.FoundOutput
		foundOutputsPerTweak, err = bip352.ReceiverScanTransaction(d.Wallet.SecretKeyScan(), d.Wallet.PubKeySpend, labelsToCheck, blockOutputs, tweak, nil)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		foundOutputs = append(foundOutputs, foundOutputsPerTweak...)
	}

	// use a map to not have to iterate for every found UTXOServed, map should be faster lookup
	matchUTXOMap := make(map[[32]byte]*networking.UTXOServed)
	for _, utxo := range utxos {
		matchUTXOMap[bip352.ConvertToFixedLength32(utxo.ScriptPubKey[2:])] = utxo
	}

	var ownedUTXOs []*wallet.OwnedUTXO
	for _, foundOutput := range foundOutputs {

		utxo, exists := matchUTXOMap[foundOutput.Output]
		if !exists {
			err = src.ErrNoMatchForUTXO
			log.Println(err)
			return nil, err
		}
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
	}

	return ownedUTXOs, err
}

func (d *Daemon) SyncToTip(chainTip uint64) error {
	var err error
	if chainTip == 0 {
		chainTip, err = d.ClientBlindBit.GetChainTip()
		if err != nil {
			log.Println(err)
			return err
		}
	}

	log.Println("Tip:", chainTip)
	// todo find fixed points for mainnet/signet/testnet where startHeight can start from. Avoid scanning through non SP merged blocks
	var startHeight = d.Wallet.BirthHeight
	if d.Wallet.LastScanHeight > startHeight {
		startHeight = d.Wallet.LastScanHeight + 1
	}

	if startHeight > chainTip {
		return nil
	}

	// don't check genesis block
	if startHeight == 0 {
		startHeight = 1
	}

	for i := startHeight; i < chainTip+1; i++ {
		go d.MarkSpentUTXOs(i)
		// possible logging here to indicate to the user
		log.Println("syncing:", i)
		var ownedUTXOs []*wallet.OwnedUTXO
		ownedUTXOs, err = d.syncBlock(i)
		if err != nil {
			log.Println(err)
			return err
		}
		if ownedUTXOs == nil {
			d.Wallet.LastScanHeight = i
			continue
		}
		err = d.Wallet.AddUTXOs(ownedUTXOs)
		if err != nil {
			log.Println(err)
			return err
		}
		d.Wallet.LastScanHeight = i

		// todo: database should be an interface to allow other forms of storing data.
		err = database.WriteToDB(config.PathDbWallet, d.Wallet)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	err = d.CheckUnspentUTXOs()
	if err != nil {
		log.Println(err)
		return err
	}
	return err
}

func (d *Daemon) ContinuousScan() error {
	d.SyncToTip(0)
	for {
		select {
		case newBlock := <-d.NewBlockChan:
			<-time.After(5 * time.Second) // delay, indexing server does not index immediately after a block is found
			oldBalance := d.Wallet.FreeBalance()
			err := d.SyncToTip(uint64(newBlock.Height))
			if err != nil {
				log.Println(err)
				return err
			}
			newBalance := d.Wallet.FreeBalance()
			if oldBalance != newBalance {
				log.Printf("New balance: %d\n", newBalance)
			}
		case height := <-d.TriggerRescanChan:
			oldBalance := d.Wallet.FreeBalance()
			err := d.ForceSyncFrom(height)
			if err != nil {
				log.Println(err)
				return err
			}
			newBalance := d.Wallet.FreeBalance()
			if oldBalance != newBalance {
				log.Printf("New balance: %d\n", newBalance)
			}
		case <-time.NewTicker(config.AutomaticScanInterval).C:
			// todo is this needed if NewBlockChan is very robust?
			// check every 5 minutes anyway
			chainTip, err := d.ClientBlindBit.GetChainTip()
			if err != nil {
				log.Println(err)
				return err
			}

			if chainTip <= d.Wallet.LastScanHeight {
				continue
			}

			err = d.SyncToTip(chainTip)
			if err != nil {
				log.Println(err)
				return err
			}
		case <-time.NewTicker(1 * time.Minute).C:
			// exclusively to check for spent UTXOs
			err := d.CheckUnspentUTXOs()
			if err != nil {
				log.Println(err)
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
		return nil
	}
	// todo this probably breaks if more than one UTXO are locked to a script
	//  this should never happen if the protocol is followed but still might occur
	for _, utxo := range d.Wallet.GetUTXOsByStates(wallet.StateUnspent, wallet.StateUnconfirmedSpent) {
		balance, err := d.ClientElectrum.GetBalance(context.Background(), utils.ConvertPubKeyToScriptHash(utxo.PubKey))
		if err != nil {
			log.Println(err)
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
		log.Println(err)
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
		log.Println(err)
		return err
	}

	if !isMatch {
		return nil
	}

	index, err := d.ClientBlindBit.GetSpentOutpointsIndex(blockHeight)
	if err != nil {
		log.Println(err)
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
		log.Println(err)
		return err
	}

	log.Printf("ForceSyncFrom: %d to %d\n", fromHeight, chainTip)

	// don't check genesis block
	if fromHeight == 0 {
		fromHeight = 1
	}

	for i := fromHeight; i < chainTip+1; i++ {
		go d.MarkSpentUTXOs(i)
		// possible logging here to indicate to the user
		log.Println("syncing:", i)
		var ownedUTXOs []*wallet.OwnedUTXO
		ownedUTXOs, err = d.syncBlock(i)
		if err != nil {
			log.Println(err)
			return err
		}
		if ownedUTXOs == nil {
			d.Wallet.LastScanHeight = i
			continue
		}
		err = d.Wallet.AddUTXOs(ownedUTXOs)
		if err != nil {
			log.Println(err)
			return err
		}
		d.Wallet.LastScanHeight = i
		err = database.WriteToDB(config.PathDbWallet, d.Wallet)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	err = d.CheckUnspentUTXOs()
	if err != nil {
		log.Println(err)
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
		log.Println(err)
		return false, err

	}

	filter, err := gcs.FromNBytes(builder.DefaultP, builder.DefaultM, nBytes)
	if err != nil {
		log.Println(err)
		return false, err
	}

	key := builder.DeriveKey(&c)

	isMatch, err := filter.HashMatchAny(key, values)
	if err != nil {
		log.Println(err)
		return false, err
	}

	if isMatch {
		return true, nil
	} else {
		return false, nil
	}
}
