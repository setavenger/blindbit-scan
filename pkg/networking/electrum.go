package networking

import (
	"context"
	"log"
	"time"

	"github.com/setavenger/go-electrum/electrum"
)

func CreateElectrumClient(address, proxy string) (*electrum.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := electrum.NewClientTCP(ctx, address, proxy)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return client, err
}
