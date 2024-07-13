package wallet

// type UTXOState int8
//
// const (
// 	StateUnconfirmed UTXOState = iota + 1
// 	StateUnspent
// 	StateSpent
// 	StateUnconfirmedSpent
// )
//
// func (u UTXOState) String() string {
// 	return [...]string{"unconfirmed", "unspent", "spent", "unconfirmed_spent"}[u-1]
// }
//
// func (u UTXOState) Index() int {
// 	return int(u)
// }
//
// func (u UTXOState) MarshalJSON() ([]byte, error) {
// 	return []byte(u.String()), nil
// }
