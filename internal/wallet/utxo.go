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

type OwnedUTXO struct {
	Txid         [32]byte      `json:"txid"`
	Vout         uint32        `json:"vout"`
	Amount       uint64        `json:"amount"`
	PrivKeyTweak [32]byte      `json:"priv_key_tweak"`
	PubKey       [32]byte      `json:"pub_key"` // are always even hence we omit the parity byte
	Timestamp    uint64        `json:"timestamp"`
	State        UTXOState     `json:"utxo_state"`
	Label        *bip352.Label `json:"label"` // the pubKey associated with the label
}

// create alias for hashes basically what btcsuite has. Better for conversion in json to hex etc.
type OwnedUtxoJSON struct {
	Txid         string           `json:"txid"`
	Vout         uint32           `json:"vout"`
	Amount       uint64           `json:"amount"`
	PrivKeyTweak string           `json:"priv_key_tweak"`
	PubKey       string           `json:"pub_key"`
	Timestamp    uint64           `json:"timestamp"`
	State        UTXOState        `json:"utxo_state"`
	Label        *Bip352LabelJSON `json:"label"` // the pubKey associated with the label
}

type Bip352LabelJSON struct {
	PubKey  string `json:"pub_key"`
	Tweak   string `json:"tweak"`
	Address string `json:"address"`
	M       uint32 `json:"m"`
}

func ConvertLabelToLabelJSON(v bip352.Label) Bip352LabelJSON {
	return Bip352LabelJSON{
		PubKey:  hex.EncodeToString(v.PubKey[:]),
		Tweak:   hex.EncodeToString(v.Tweak[:]),
		Address: v.Address,
		M:       v.M,
	}
}

func ConvertLabelJSONToLabel(v Bip352LabelJSON) (*bip352.Label, error) {
	pubKey, err := hex.DecodeString(v.PubKey)
	if err != nil {
		return nil, err
	}
	tweak, err := hex.DecodeString(v.Tweak)
	if err != nil {
		return nil, err
	}
	label := &bip352.Label{
		PubKey:  bip352.ConvertToFixedLength33(pubKey),
		Tweak:   bip352.ConvertToFixedLength32(tweak),
		Address: v.Address,
		M:       v.M,
	}

	return label, err
}

func (u OwnedUTXO) MarshalJSON() ([]byte, error) {

	var label *Bip352LabelJSON
	if u.Label != nil {
		label = &Bip352LabelJSON{
			PubKey:  hex.EncodeToString(u.Label.PubKey[:]),
			Tweak:   hex.EncodeToString(u.Label.Tweak[:]),
			Address: u.Label.Address,
			M:       u.Label.M,
		}
	}
	newUtxo := OwnedUtxoJSON{
		Txid:         hex.EncodeToString(u.Txid[:]),
		Vout:         u.Vout,
		Amount:       u.Amount,
		PrivKeyTweak: hex.EncodeToString(u.PrivKeyTweak[:]),
		PubKey:       hex.EncodeToString(u.PubKey[:]),
		Timestamp:    u.Timestamp,
		State:        u.State,
		Label:        label,
	}

	return json.Marshal(newUtxo)
}

func (u *OwnedUTXO) UnmarshalJSON(data []byte) error {
	var aux OwnedUtxoJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	txid, err := hex.DecodeString(aux.Txid)
	if err != nil {
		return err
	}
	privKeyTweak, err := hex.DecodeString(aux.PrivKeyTweak)
	if err != nil {
		return err
	}
	pubKey, err := hex.DecodeString(aux.PubKey)
	if err != nil {
		return err
	}

	var label *bip352.Label
	if aux.Label != nil {
		label, err = ConvertLabelJSONToLabel(*aux.Label)
		if err != nil {
			return err
		}
	}

	*u = OwnedUTXO{
		Txid:         bip352.ConvertToFixedLength32(txid),
		Vout:         aux.Vout,
		Amount:       aux.Amount,
		PrivKeyTweak: bip352.ConvertToFixedLength32(privKeyTweak),
		PubKey:       bip352.ConvertToFixedLength32(pubKey),
		Timestamp:    aux.Timestamp,
		State:        aux.State,
		Label:        label,
	}
	return err
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
