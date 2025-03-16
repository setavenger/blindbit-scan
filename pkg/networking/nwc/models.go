package nwc

import (
	"encoding/json"

	"github.com/setavenger/blindbit-scan/pkg/logging"
	"github.com/setavenger/blindbit-scan/pkg/wallet"
)

const (
	// selected request methods
	GET_INFO_METHOD          = "get_info"
	GET_BALANCE_METHOD       = "get_balance"
	LIST_UTXOS_METHOD        = "list_utxos"
	LIST_TRANSACTIONS_METHOD = "list_transactions"
)

type Nip47Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type Nip47Response struct {
	ResultType string          `json:"result_type"`
	Error      ErrorBody       `json:"error"`
	Result     json.RawMessage `json:"result"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type GetBalanceResponseBody struct {
	Balance int64 `json:"balance"` // in millisatoshis
}

// GetInfoResponse defines the structure for a get_info response.
type GetInfoResponseBody struct {
	Alias       string   `json:"alias"`
	PubKey      string   `json:"pubkey"`
	Network     string   `json:"network"`
	BlockHeight int      `json:"block_height"`
	Methods     []string `json:"methods"`
}

type ListUtxosResponseBody struct {
	Utxos wallet.UtxoCollection `json:"utxos"`
}

// marshals into the passed request struct
// if fails returns a error repsonse
func decodeRequest(request *Nip47Request, methodParams any) *Nip47Response {
	err := json.Unmarshal(request.Params, methodParams)
	if err != nil {
		logging.L.Err(err).Any("request", request).Msg("Failed to decode NIP-47 request")
		return &Nip47Response{
			ResultType: request.Method,
			Error: ErrorBody{
				Code:    "BAD_REQUEST",
				Message: err.Error(),
			}}
	}
	return nil
}
