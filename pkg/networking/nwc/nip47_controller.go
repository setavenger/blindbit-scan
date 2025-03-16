package nwc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip04"
	"github.com/setavenger/blindbit-scan/pkg/logging"
)

const (
	relayURL string = "wss://relay.getalby.com/v1"
)

type Nip47Controller struct {
	ctx   context.Context
	relay *nostr.Relay
	// handler maps methods to handler funcs
	handlers          map[string]Nip47ControllerHandlerFunc // place holder for now
	apps              Apps
	stopListeningChan chan struct{}
}

func (c *Nip47Controller) Apps() Apps {
	return c.apps
}

func NewNip47Controller(ctx context.Context) *Nip47Controller {
	return &Nip47Controller{
		ctx:               ctx,
		relay:             &nostr.Relay{},
		handlers:          map[string]Nip47ControllerHandlerFunc{},
		apps:              map[string]AppsItem{},
		stopListeningChan: make(chan struct{}),
	}
}

func NewNip47ControllerFromApps(ctx context.Context, apps Apps) *Nip47Controller {
	return &Nip47Controller{
		ctx:               ctx,
		relay:             &nostr.Relay{},
		handlers:          map[string]Nip47ControllerHandlerFunc{},
		apps:              apps,
		stopListeningChan: make(chan struct{}),
	}
}

func (c *Nip47Controller) ConnectRelay() error {
	relay, err := nostr.RelayConnect(c.ctx, relayURL)
	if err != nil {
		logging.L.Fatal().Err(err).Msg("Failed to connect to relay")
		return err
	}

	c.relay = relay

	return nil
}

func (c *Nip47Controller) RegisterHandler(
	method string,
	handler Nip47ControllerHandlerFunc,
) {
	c.handlers[method] = handler
}

// handles a decoded Nip47Request
// must return the raw marshalled response as byte slice for later encrypting and publishing
type Nip47ControllerHandlerFunc func(context.Context, Nip47Request) ([]byte, error)

func (c *Nip47Controller) NewConnection() (
	pubKeyWalletService string,
	secretClient string,
	err error,
) {
	privKeyWalletService := nostr.GeneratePrivateKey()
	pubKeyWalletService, err = nostr.GetPublicKey(privKeyWalletService)
	if err != nil {
		return
	}

	secretClient = nostr.GeneratePrivateKey()
	pubKeyClient, err := nostr.GetPublicKey(secretClient)
	if err != nil {
		return
	}

	newKeystore := AppsItem{
		WalletPriv: privKeyWalletService,
		WalletPub:  pubKeyWalletService,
		ClientPub:  pubKeyClient,
	}

	err = c.PublishInfoEvent(c.relay, privKeyWalletService, pubKeyWalletService)
	if err != nil {
		return
	}

	c.apps[pubKeyClient] = newKeystore

	// stop and restart listening so new app is being listened for
	c.StopListening()
	go c.StartListening()

	return
}
func (c *Nip47Controller) StopListening() {
	c.stopListeningChan <- struct{}{}
}

// NewConnectionUri calls NewConnection but simply returns the uri and a possible error
func (c *Nip47Controller) NewConnectionUri() (uri string, err error) {
	pubKeyWalletService, clientSecret, err := c.NewConnection()
	uri = fmt.Sprintf(
		"nostr+walletconnect://%s?relay=%s&secret=%s",
		pubKeyWalletService, relayURL, clientSecret,
	)
	return
}

// PublishInfoEvent publishes the initial replaceable info event (kind 13194) to the relay.
func (c *Nip47Controller) PublishInfoEvent(
	relay *nostr.Relay,
	privKey, pubKey string,
) error {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	// Supported commands as a space-separated string.
	// Use simple and standard to blend in
	infoContent := "get_info get_balance"
	infoEvent := nostr.Event{
		Kind:      13194,
		Content:   infoContent,
		CreatedAt: nostr.Now(),
		PubKey:    pubKey,
	}
	if err := infoEvent.Sign(privKey); err != nil {
		log.Printf("Error signing info event: %v", err)
		return err
	}

	err := relay.Publish(ctx, infoEvent)
	if err != nil {
		log.Printf("Failed to publish info event: %v", err)
		return err
	}

	return nil
}

func (c *Nip47Controller) StartListening() {
	sub, err := c.relay.Subscribe(c.ctx, c.buildFilters())
	if err != nil {
		logging.L.Fatal().Err(err).Msg("Failed to subscribe to relay")
	}
	logging.L.Info().Msg("Subscribed to relay events. Waiting for requests...")

	for i := range c.apps {
		logging.L.Info().Msgf("listening for app: %s", c.apps.FindByClientPub(i).WalletPub)
	}

	for {
		select {
		case ev := <-sub.Events:
			logging.L.Info().Str("event-id", ev.ID).Msg("received event")
			go c.processEvent(ev)
		case <-c.ctx.Done():
			sub.Unsub()
			logging.L.Info().Msg("Nip47Controller context done")
			return
		case <-c.stopListeningChan:
			sub.Unsub()
			logging.L.Info().Msg("unsubscribed from events")
			return
		}
	}
}

func (c *Nip47Controller) processEvent(ev *nostr.Event) {
	req, err := c.DecryptEvent(ev)
	if err != nil {
		logging.L.Err(err).Any("event", ev).Msg("failed to decrypt event")
		// c.publishErrorResponse(
		// 	app, ev, req.Method, "NOT_IMPLEMENTED",
		// 	fmt.Errorf("method not implemented: %s", req.Method),
		// )
		return
	}

	// Find the app corresponding to the client public key.
	app := c.apps.FindByClientPub(ev.PubKey)
	if app == nil {
		logging.L.Warn().Msgf("no app found for pubkey: %s", ev.PubKey)
		c.publishErrorResponse(
			app, ev, req.Method, "UNAUTHORIZED",
			fmt.Errorf("no app for: %s", ev.PubKey),
		)
		return
	}

	// Lookup the handler function based on the method.
	handlerFunc, ok := c.handlers[req.Method]
	if !ok {
		logging.L.Error().Msgf("no handler for method: %s", req.Method)
		c.publishErrorResponse(
			app, ev, req.Method, "NOT_IMPLEMENTED",
			fmt.Errorf("method not implemented: %s", req.Method),
		)
		return
	}

	// Execute the handler to get the response bytes.
	respData, err := handlerFunc(c.ctx, req)
	if err != nil {
		logging.L.Err(err).Any("request", req).Msg("error in handlerFunc")
		c.publishErrorResponse(app, ev, req.Method, "INTERNAL", err)
		return
	}

	// Publish the response.
	c.publishResponse(app, ev, respData)
}

func (c Nip47Controller) DecryptEvent(ev *nostr.Event) (req Nip47Request, err error) {
	var plainText string
	plainText, err = c.DecryptEventToPlainText(ev)
	if err != nil {
		logging.L.Err(err).Msg("failed to decrypt event")
		return
	}

	if err = json.Unmarshal([]byte(plainText), &req); err != nil {
		logging.L.Err(err).Any("event", ev).Msg("Failed to unmarshal event content")
		return
	}
	return
}

func (c Nip47Controller) DecryptEventToPlainText(
	ev *nostr.Event,
) (
	plainText string,
	err error,
) {
	app := c.apps.FindByClientPub(ev.PubKey)
	if app == nil {
		err = fmt.Errorf("no app found for %s", ev.PubKey)
		return
	}

	ss, err := nip04.ComputeSharedSecret(ev.PubKey, app.WalletPriv)
	if err != nil {
		logging.L.Err(err).Any("event", ev).Msg("Failed to generate secret key")
		return
	}

	plainText, err = nip04.Decrypt(ev.Content, ss)
	if err != nil {
		logging.L.Err(err).Any("event", ev).Msg("decrypt error")
		return
	}
	return
}

func (c Nip47Controller) publishResponse(
	app *AppsItem,
	reqEvent *nostr.Event,
	contentBytes []byte,
) {
	ss, err := nip04.ComputeSharedSecret(app.ClientPub, app.WalletPriv)
	if err != nil {
		logging.L.Err(err).Msg("Failed to generate shared secret key")
		return
	}
	encryptedData, err := nip04.Encrypt(string(contentBytes), ss)
	if err != nil {
		logging.L.Err(err).Msg("Failed to encrypt data")
		return
	}

	respEvent := nostr.Event{
		Kind:      23195, // response event kind
		Content:   encryptedData,
		CreatedAt: nostr.Now(),
		Tags:      nostr.Tags{{"p", app.ClientPub}, {"e", reqEvent.ID}},
		PubKey:    app.WalletPub,
	}
	if err := respEvent.Sign(app.WalletPriv); err != nil {
		logging.L.Err(err).Msg("Error signing response event")
		return
	}

	err = c.relay.Publish(c.ctx, respEvent)
	if err != nil {
		logging.L.Err(err).Msg("Error publishing response event")
		return
	}
}

func (c Nip47Controller) publishErrorResponse(
	app *AppsItem,
	reqEvent *nostr.Event,
	reqMethod string,
	errorCode string,
	err error,
) {
	resp := &Nip47Response{
		ResultType: reqMethod,
		Error: ErrorBody{
			Code:    errorCode,
			Message: err.Error(),
		}}
	contentBytes, err := json.Marshal(resp)
	if err != nil {
		logging.L.Err(err).Msg("could not marshal error response")
		return
	}
	c.publishResponse(app, reqEvent, contentBytes)
}

func (c Nip47Controller) buildFilters() nostr.Filters {
	var filters nostr.Filters
	for _, pub := range c.apps.AllWalletServicePubs() {
		filters = append(filters, nostr.Filter{
			Kinds: []int{23194},
			Tags:  map[string][]string{"p": {pub}},
		})
	}
	return filters
}

// for now we use it as a proxy
// only the apps data can and should be stored

func (c *Nip47Controller) Serialise() ([]byte, error) {
	return c.apps.Serialise()
}

func (c *Nip47Controller) DeSerialise(data []byte) error {
	return c.apps.DeSerialise(data)
}
