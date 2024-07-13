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
	StateUnconfirmedSpent
	StateSpent
)

func (u UTXOState) String() string {
	return [...]string{"unconfirmed", "unspent", "spent", "unconfirmed_spent"}[u-1]
}

func (u UTXOState) Index() int {
	return int(u)
}

// func (u UTXOState) MarshalJSON() ([]byte, error) {
// 	val := u.String()
// 	log.Println(val)
// 	log.Printf("%+v\n", val)
// 	return []byte(val), nil
// }

type OwnedUTXO struct {
	Txid         [32]byte      `json:"txid"`
	Vout         uint32        `json:"vout"`
	Amount       uint64        `json:"amount"`
	PrivKeyTweak [32]byte      `json:"priv_key_tweak"`
	PubKey       [32]byte      `json:"pub_key"`
	Timestamp    uint64        `json:"timestamp"`
	State        UTXOState     `json:"utxo_state"`
	Label        *bip352.Label `json:"label"` // the pubKey associated with the label
}

// create alias for hashes basically what btcsuite has. Better for conversion in json to hex etc.
type OwnedUtxoJSON struct {
	Txid         string          `json:"txid"`
	Vout         uint32          `json:"vout"`
	Amount       uint64          `json:"amount"`
	PrivKeyTweak string          `json:"priv_key_tweak"`
	PubKey       string          `json:"pub_key"`
	Timestamp    uint64          `json:"timestamp"`
	State        UTXOState       `json:"utxo_state"`
	Label        Bip352LabelJSON `json:"label"` // the pubKey associated with the label
}

type Bip352LabelJSON struct {
	PubKey  string `json:"pub_key"`
	Tweak   string `json:"tweak"`
	Address string `json:"address"`
	M       uint32 `json:"m"`
}

func (s OwnedUTXO) MarshalJSON() ([]byte, error) {
	newUtxo := OwnedUtxoJSON{
		Txid:         hex.EncodeToString(s.Txid[:]),
		Vout:         s.Vout,
		Amount:       s.Amount,
		PrivKeyTweak: hex.EncodeToString(s.PrivKeyTweak[:]),
		PubKey:       hex.EncodeToString(s.PubKey[:]),
		Timestamp:    s.Timestamp,
		State:        s.State,
		Label: Bip352LabelJSON{
			PubKey:  hex.EncodeToString(s.Label.PubKey[:]),
			Tweak:   hex.EncodeToString(s.Label.Tweak[:]),
			Address: s.Label.Address,
			M:       s.Label.M,
		},
	}

	return json.Marshal(newUtxo)
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
