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
