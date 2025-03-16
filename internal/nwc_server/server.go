package nwcserver

import (
	"context"
	"encoding/json"

	"github.com/setavenger/blindbit-scan/internal/config"
	"github.com/setavenger/blindbit-scan/internal/daemon"
	"github.com/setavenger/blindbit-scan/pkg/logging"
	"github.com/setavenger/blindbit-scan/pkg/networking/nwc"
	"github.com/setavenger/blindbit-scan/pkg/wallet"
)

type NwcServer struct {
	Daemon *daemon.Daemon
}

func NewNwcServer(d *daemon.Daemon) *NwcServer {
	return &NwcServer{Daemon: d}
}

func (s *NwcServer) GetInfoHandler() nwc.Nip47ControllerHandlerFunc {
	return func(ctx context.Context, nr nwc.Nip47Request) (data []byte, err error) {
		rawData := nwc.GetInfoResponseBody{
			Alias:       "", //todo: is this relevant?
			PubKey:      "", // todo: find a way to pass this along. maybe via the, currently unsused, context
			Network:     config.ChainParams.Name,
			BlockHeight: int(s.Daemon.Wallet.LastScanHeight),
			Methods:     []string{"get_info", "get_balance", "list_utxos"},
		}
		var resultData []byte
		resultData, err = json.Marshal(rawData)
		if err != nil {
			logging.L.Err(err).Msg("could not raw data")
			return
		}
		resp := nwc.Nip47Response{
			ResultType: nwc.GET_BALANCE_METHOD,
			Error:      nwc.ErrorBody{},
			Result:     resultData,
		}

		data, err = json.Marshal(resp)
		if err != nil {
			logging.L.Err(err).Msg("could not marshal Nip47Response")
			return
		}
		return
	}
}

func (s *NwcServer) GetBalanceHandler() nwc.Nip47ControllerHandlerFunc {
	return func(ctx context.Context, nr nwc.Nip47Request) (data []byte, err error) {
		rawData := nwc.GetBalanceResponseBody{
			Balance: int64(s.Daemon.Wallet.FreeBalance()),
		}
		var resultData []byte
		resultData, err = json.Marshal(rawData)
		if err != nil {
			logging.L.Err(err).Msg("could not marshal raw data")
			return
		}
		resp := nwc.Nip47Response{
			ResultType: nwc.GET_BALANCE_METHOD,
			Error:      nwc.ErrorBody{},
			Result:     resultData,
		}

		data, err = json.Marshal(resp)
		if err != nil {
			logging.L.Err(err).Msg("could not marshal Nip47Response")
			return
		}
		return
	}
}

func (s *NwcServer) ListUtxosHandler() nwc.Nip47ControllerHandlerFunc {
	return func(ctx context.Context, nr nwc.Nip47Request) (data []byte, err error) {
		utxos := make(wallet.UtxoCollection, len(s.Daemon.Wallet.UTXOs))
		copy(utxos, s.Daemon.Wallet.UTXOs)
		rawData := nwc.ListUtxosResponseBody{
			Utxos: utxos,
		}
		var resultData []byte
		resultData, err = json.Marshal(rawData)
		if err != nil {
			logging.L.Err(err).Msg("could not raw data")
			return
		}
		resp := nwc.Nip47Response{
			ResultType: nwc.LIST_UTXOS_METHOD,
			Error:      nwc.ErrorBody{},
			Result:     resultData,
		}

		data, err = json.Marshal(resp)
		if err != nil {
			logging.L.Err(err).Msg("could not marshal Nip47Response")
			return
		}
		return
	}
}
