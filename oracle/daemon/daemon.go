package daemon

import (
	"context"
	"fmt"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/encoding"
	"github.com/GPTx-global/guru/oracle/subscribe"
	"github.com/GPTx-global/guru/oracle/tx"
	"github.com/GPTx-global/guru/oracle/types"
	"github.com/GPTx-global/guru/oracle/woker"
	"github.com/cosmos/cosmos-sdk/client"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/tendermint/tendermint/rpc/client/http"
)

type Daemon struct {
	client    *http.HTTP
	clientCtx client.Context

	subscribeManager   *subscribe.SubscribeManager
	jobManager         *woker.JobManager
	transactionManager *tx.TxManager

	ctx context.Context
}

// New creates a new Oracle daemon instance with initialized components
func New(ctx context.Context) (*Daemon, error) {
	d := new(Daemon)
	d.ctx = ctx

	clt, err := http.New(types.Config.RpcEndpoint(), "/websocket")
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	d.client = clt

	keyRing, err := types.Config.Keyring()
	if err != nil {
		return nil, fmt.Errorf("failed to create keyring: %w", err)
	}

	key, err := keyRing.Key(types.Config.KeyName())
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	fromAddress, err := key.GetAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get from address: %w", err)
	}

	encCfg := encoding.MakeConfig(app.ModuleBasics)
	d.clientCtx = client.Context{}.
		WithCodec(encCfg.Codec).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithTxConfig(encCfg.TxConfig).
		WithLegacyAmino(encCfg.Amino).
		WithKeyring(keyRing).
		WithChainID(types.Config.ChainID()).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithNodeURI(types.Config.RpcEndpoint()).
		WithClient(d.client).
		WithFromAddress(fromAddress).
		WithFromName(types.Config.KeyName()).
		WithBroadcastMode("sync")

	d.transactionManager = tx.NewTxManager(d.clientCtx)
	d.subscribeManager = subscribe.NewSubscribeManager(d.ctx)
	d.jobManager = woker.NewJobManager()
	return d, nil
}

// Start initializes and starts all daemon components including job manager, client, and event subscriptions
func (d *Daemon) Start() error {
	d.jobManager.Start(d.ctx, d.transactionManager.ResultQueue())

	err := d.client.Start()
	if err != nil {
		return fmt.Errorf("failed to start client: %w", err)
	}

	docs, err := d.subscribeManager.LoadRegisterRequest(d.clientCtx)
	if err != nil {
		return fmt.Errorf("failed to load register request: %w", err)
	}
	for _, doc := range docs {
		if jobs := types.MakeJobs(doc); jobs != nil {
			d.ProcessJob(jobs)
		}
	}

	err = d.subscribeManager.SetSubscribe(d.client)
	if err != nil {
		return fmt.Errorf("failed to set subscribe: %w", err)
	}

	return nil
}

// Stop gracefully shuts down all daemon components
func (d *Daemon) Stop() {
	d.jobManager.Stop()
	d.client.Stop()
	d.ctx.Done()
}

// Monitor continuously listens for new events and processes them as jobs
func (d *Daemon) Monitor() {
	for {
		if jobs := d.subscribeManager.Subscribe(); jobs != nil {
			d.ProcessJob(jobs)
		}
	}
}

// ProcessJob submits a job to the job manager for execution
func (d *Daemon) ProcessJob(jobs []*types.Job) {
	for _, job := range jobs {
		d.jobManager.SubmitJob(job)
	}
}

// ServeOracle continuously builds and broadcasts oracle data submission transactions
func (d *Daemon) ServeOracle() error {
	for {
		txBytes, err := d.transactionManager.BuildSubmitTx()
		if err != nil {
			return fmt.Errorf("failed to build submit tx: %w", err)
		}

		txResponse, err := d.transactionManager.BroadcastTx(txBytes)
		if err != nil {
			return fmt.Errorf("failed to broadcast tx: %w", err)
		}

		if txResponse.Code == 0 {
			fmt.Printf(", Hash: %s\n", txResponse.TxHash)
			d.transactionManager.IncrementSequenceNumber()
		} else {
			fmt.Printf("tx failed: %s\n", txResponse.RawLog)
		}
	}
}
