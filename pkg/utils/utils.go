package utils

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/setavenger/go-bip352"
)

// ConvertPubKeyToScriptHash
// Converts the given taproot pubKey to a scriptHash which can be checked with electrumX
func ConvertPubKeyToScriptHash(pubKey [32]byte) string {
	data := append([]byte{0x51, 0x20}, pubKey[:]...)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(bip352.ReverseBytesCopy(hash[:]))
}
