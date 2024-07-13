package wallet

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/setavenger/go-bip352"
)

type LabelMap map[[33]byte]*bip352.Label

func (lm *LabelMap) MarshalJSON() ([]byte, error) {
	// Convert map to a type that can be marshaled by the standard JSON package
	aux := make(map[string]*bip352.Label)
	for k, v := range *lm {
		key := fmt.Sprintf("%x", k) // Convert byte array to hex string
		aux[key] = v
	}
	return json.Marshal(aux)
}

func (lm *LabelMap) UnmarshalJSON(data []byte) error {
	aux := make(map[string]*bip352.Label)
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
		(*lm)[key] = v
	}
	return nil
}
