package wallet

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"github.com/setavenger/go-bip352"
)

type UTXOState int8

const (
	StateUnconfirmed UTXOState = iota + 1
	StateUnspent
	StateSpent
	StateUnconfirmedSpent
)

func (u UTXOState) String() string {
	return [...]string{"unconfirmed", "unspent", "spent", "unconfirmed_spent"}[u-1]
}

func (u UTXOState) Index() int {
	return int(u)
}

func (u UTXOState) MarshalJSON() ([]byte, error) {
	return []byte(u.String()), nil
}

type OwnedUTXO struct {
	Txid         [32]byte      `json:"txid,omitempty"`
	Vout         uint32        `json:"vout,omitempty"`
	Amount       uint64        `json:"amount"`
	PrivKeyTweak [32]byte      `json:"priv_key_tweak,omitempty"`
	PubKey       [32]byte      `json:"pub_key,omitempty"`
	Timestamp    uint64        `json:"timestamp,omitempty"`
	State        UTXOState     `json:"utxo_state,omitempty"`
	Label        *bip352.Label `json:"label"` // the pubKey associated with the label
}

func (u OwnedUTXO) SerialiseToOutpoint() ([36]byte, error) {
	var buf bytes.Buffer
	buf.Write(bip352.ReverseBytesCopy(u.Txid[:]))
	err := binary.Write(&buf, binary.LittleEndian, u.Vout)
	if err != nil {
		log.Println(err)
		return [36]byte{}, err
	}

	var outpoint [36]byte
	copy(outpoint[:], buf.Bytes())
	return outpoint, nil
}

func (u *OwnedUTXO) LabelPubKey() []byte {
	if u.Label != nil {
		return u.Label.PubKey[:]
	} else {
		return nil
	}
}

func (u *OwnedUTXO) GetKey() ([36]byte, error) {
	var buf bytes.Buffer
	buf.Write(u.Txid[:])
	err := binary.Write(&buf, binary.BigEndian, u.Vout)
	if err != nil {
		log.Println(err)
		return [36]byte{}, err
	}

	var result [36]byte
	copy(result[:], buf.Bytes())

	return result, nil
}

type UtxoCollection []*OwnedUTXO

// UTXOMapping
// the key is the utxos (txid||vout)
// todo marshalling or unmarshalling seems to have some issues. Investigate root cause.
type UTXOMapping map[[36]byte]struct{}

func (um *UTXOMapping) MarshalJSON() ([]byte, error) {
	// Convert map to a type that can be marshaled by the standard JSON package
	aux := make(map[string]struct{})
	for k, v := range *um {
		key := fmt.Sprintf("%x", k) // Convert byte array to hex string
		aux[key] = v
	}
	return json.Marshal(aux)
}

func (um *UTXOMapping) UnmarshalJSON(data []byte) error {
	aux := make(map[string]struct{})
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	*um = make(UTXOMapping)
	for k, v := range aux {
		var key [36]byte
		_, err := hex.Decode(key[:], []byte(k))
		if err != nil {
			return err
		}
		(*um)[key] = v
	}
	return nil
}
