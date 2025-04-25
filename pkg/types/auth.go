package types

import "encoding/json"

type AuthCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *AuthCredentials) Serialise() ([]byte, error) {
	return json.Marshal(a)
}

func (a *AuthCredentials) DeSerialise(data []byte) error {
	return json.Unmarshal(data, a)
}
