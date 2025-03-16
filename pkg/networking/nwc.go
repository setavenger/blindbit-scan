package networking

import (
	"context"
	"fmt"
	"log"

	"github.com/nbd-wtf/go-nostr"
)

// GetInfoResponse defines the structure for a get_info response.
type GetInfoResponse struct {
	Alias       string   `json:"alias"`
	PubKey      string   `json:"pubkey"`
	Network     string   `json:"network"`
	BlockHeight int      `json:"block_height"`
	Methods     []string `json:"methods"`
}

type Nip47Response struct {
	ResultType string         `json:"result_type"`
	Error      Nip47ErrorBody `json:"error"`
	Result     any            `json:"result"`
}

type Nip47ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// GetBalanceResponse defines the structure for a get_balance response.
type GetBalanceResponse struct {
	Balance int64 `json:"balance"` // in millisatoshis
}

const (
	privKeyWalletService = "66d1069f89c77d846cca006d132f2493a0acd31f650e000bc5199306556d8b21"
	pubKeyWalletService  = "3e6221b76a6555c819c8d0c87d9af47ce1b31c0b413f276b2f0d8ddd2a3e1c1f"
	privKeyClientSecret  = "125b81e4b8b21c889374ebf1a4b786791f0088a4f3b64260e75fd759497c2700"
	pubKeyClient         = "e89262ebbe2b114bfa1f53b0b65a46a9bdf65ac2c11a7969933c9aac8a4e83dd"
)

// publishInfoEvent publishes a replaceable info event (kind 13194) to the relay.
func publishInfoEvent(ctx context.Context, relay *nostr.Relay, privKey, pubKey string) {
	// Supported commands as a space-separated string.
	infoContent := "get_info get_balance"
	infoEvent := nostr.Event{
		Kind:      13194,
		Content:   infoContent,
		CreatedAt: nostr.Now(),
		PubKey:    pubKey,
		// You can add additional tags like "notifications" if needed.
	}
	if err := infoEvent.Sign(privKey); err != nil {
		log.Printf("Error signing info event: %v", err)
		return
	}

	err := relay.Publish(ctx, infoEvent)
	if err != nil {
		log.Printf("Failed to publish info event: %v", err)
	}
	fmt.Println("Successfully broadcasted")
}
