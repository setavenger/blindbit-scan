package nwc

import "encoding/json"

// Apps maps client pubkey to wallet key data
type Apps map[string]AppsItem

type AppsItem struct {
	Name       string
	WalletPriv string
	WalletPub  string
	ClientPub  string
	// Potentially add metadata
}

func (ks Apps) FindByClientPub(pub string) *AppsItem {
	if item, ok := ks[pub]; ok {
		return &item
	}
	return nil
}

func (ks Apps) FindByWalletServicePub(pub string) *AppsItem {
	for key := range ks {
		val := ks[key]
		if val.WalletPub == pub {
			return &val
		}
	}

	return nil
}

func (ks Apps) AllWalletServicePubs() []string {
	var out []string
	for key := range ks {
		val := ks[key]
		out = append(out, val.WalletPub)
	}
	return out
}

func (a *Apps) Serialise() ([]byte, error) {
	return json.Marshal(a)
}

func (a *Apps) DeSerialise(data []byte) error {
	return json.Unmarshal(data, a)
}
