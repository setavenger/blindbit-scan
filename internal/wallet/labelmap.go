package wallet

import (
	"encoding/hex"
	"encoding/json"

	"github.com/setavenger/blindbit-scan/pkg/types"
	"github.com/setavenger/go-bip352"
)

type LabelMap map[types.PublicKey]*bip352.Label

func (lm *LabelMap) MarshalJSON() ([]byte, error) {
	// Convert map to a type that can be marshaled by the standard JSON package
	aux := make(map[string]Bip352LabelJSON)
	for k, v := range *lm {
		aux[k.String()] = Bip352LabelJSON{
			PubKey:  hex.EncodeToString(v.PubKey[:]),
			Tweak:   hex.EncodeToString(v.Tweak[:]),
			Address: v.Address,
			M:       v.M,
		}
	}
	return json.Marshal(aux)
}

func (lm *LabelMap) UnmarshalJSON(data []byte) error {
	aux := make(map[string]Bip352LabelJSON)
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	*lm = make(LabelMap)
	for k, v := range aux {
		var key [33]byte
		_, err := hex.Decode(key[:], []byte(k))
		if err != nil {
			return err
		}
		label, err := ConvertLabelJSONToLabel(v)
		if err != nil {
			return err
		}

		(*lm)[key] = label
	}
	return nil
}
