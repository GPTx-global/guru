package daemon

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/encoding"
	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
	"github.com/GPTx-global/guru/oracle/monitor"
	"github.com/GPTx-global/guru/oracle/scheduler"
	"github.com/GPTx-global/guru/oracle/submitter"
	"github.com/GPTx-global/guru/oracle/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

type Daemon struct {
	ctx       context.Context
	client    *http.HTTP
	clientCtx client.Context

	monitor   *monitor.Monitor
	scheduler *scheduler.Scheduler
	submitter *submitter.Submitter
}

// New creates a new Oracle daemon instance with initialized components
func New(ctx context.Context) (*Daemon, error) {
	d := new(Daemon)
	d.ctx = ctx

	clt, err := http.New(config.ChainEndpoint(), "/websocket")
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	d.client = clt

	encCfg := encoding.MakeConfig(app.ModuleBasics)
	d.clientCtx = client.Context{}.
		WithCodec(encCfg.Codec).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithTxConfig(encCfg.TxConfig).
		WithLegacyAmino(encCfg.Amino).
		WithKeyring(config.Keyring()).
		WithChainID(config.ChainID()).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithNodeURI(config.ChainEndpoint()).
		WithClient(d.client).
		WithFromAddress(config.Address()).
		WithFromName(config.KeyName()).
		WithBroadcastMode("block") // sync

	d.monitor = monitor.New(d.clientCtx, d.ctx)
	d.scheduler = scheduler.New()
	d.submitter = submitter.New(d.clientCtx)

	return d, nil
}

// Start initializes and starts all daemon components including job manager, client, and event subscriptions
func (d *Daemon) Start() error {
	err := d.client.Start()
	if err != nil {
		return fmt.Errorf("failed to start client: %w", err)
	}

	d.monitor.Start()
	d.scheduler.Start()

	go d.Monitor()
	go d.ServeResult()

	return nil
}

// Stop gracefully shuts down all daemon components
func (d *Daemon) Stop() {
	d.monitor.Stop()
	d.scheduler.Stop()
	d.client.Stop()
}

// Monitor continuously listens for new events and processes them as jobs
func (d *Daemon) Monitor() {
	docs, err := d.monitor.LoadRequestDocs()
	if err != nil {
		log.Errorf("failed to load request docs: %v", err)
	}

	for _, doc := range docs {
		if slices.Contains(doc.AccountList, d.clientCtx.GetFromAddress().String()) {
			err := d.scheduler.ProcessRequestDoc(*doc)
			if err != nil {
				log.Errorf("failed to process request doc: %v", err)
			}
		}
	}

	for {
		oracleEvent := d.monitor.Subscribe()
		if oracleEvent == nil {
			continue
		}

		switch oracleEvent := oracleEvent.(type) {
		case oracletypes.OracleRequestDoc:
			err := d.scheduler.ProcessRequestDoc(oracleEvent)
			if err != nil {
				log.Errorf("failed to process request doc: %v", err)
			}
		case coretypes.ResultEvent:
			if gasPrices, ok := oracleEvent.Events[types.MinGasPrice]; ok {
				for _, gasPrice := range gasPrices {
					config.SetGasPrice(gasPrice + "aguru")
				}
			}

			for i, id := range oracleEvent.Events[types.CompleteID] {
				nonce, err := strconv.ParseUint(oracleEvent.Events[types.CompleteNonce][i], 10, 64)
				if err != nil {
					log.Errorf("failed to parse nonce: %v", err)
					continue
				}

				err = d.scheduler.ProcessComplete(id, nonce)
				if err != nil {
					log.Errorf("failed to process complete: %v", err)
				}
			}
		}
	}
}

func (d *Daemon) ServeResult() {
	for result := range d.scheduler.Results() {
		factory, txBuilder, err := d.submitter.BuildTransaction(result)
		if err != nil {
			log.Errorf("failed to build transaction: %v", err)
			continue
		}

		txBytes, err := d.submitter.SignTransaction(factory, txBuilder)
		if err != nil {
			log.Errorf("failed to sign transaction: %v", err)
			continue
		}

		err = d.submitter.BroadcastTransaction(txBytes)
		if err != nil {
			log.Errorf("failed to broadcast transaction: %v", err)
		}
	}
}
