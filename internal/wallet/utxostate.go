package wallet

import (
	"fmt"
	"strings"
)

type UTXOState int8

const (
	StateUnconfirmed UTXOState = iota + 1
	StateUnspent
	StateUnconfirmedSpent
	StateSpent
)

func (u UTXOState) String() string {
	return [...]string{"unconfirmed", "unspent", "unconfirmed_spent", "spent"}[u-1]
}

func (u UTXOState) Index() int {
	return int(u)
}

func (u UTXOState) MarshalJSON() ([]byte, error) {
	val := u.String()
	return []byte("\"" + val + "\""), nil
}

func (u *UTXOState) UnmarshalJSON(data []byte) error {
	switch strings.ReplaceAll(string(data), "\"", "") {
	case StateUnconfirmed.String():
		*u = StateUnconfirmed
	case StateUnspent.String():
		*u = StateUnspent
	case StateUnconfirmedSpent.String():
		*u = StateUnconfirmedSpent
	case StateSpent.String():
		*u = StateSpent
	default:
		return fmt.Errorf("err: %s is not a valid state", string(data))
	}
	return nil
}
