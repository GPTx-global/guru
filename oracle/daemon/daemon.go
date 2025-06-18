package daemon

import (
	"context"
	"fmt"
	"slices"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/encoding"
	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
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

	clt, err := http.New(config.ChainEndpoint(), "/websocket")
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	d.client = clt

	keyRing := config.Keyring()
	key, err := keyRing.Key(config.KeyName())
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
		WithChainID(config.ChainID()).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithNodeURI(config.ChainEndpoint()).
		WithClient(d.client).
		WithFromAddress(fromAddress).
		WithFromName(config.KeyName()).
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
label:
	for _, doc := range docs {
		if !slices.Contains(doc.AccountList, d.clientCtx.GetFromAddress().String()) {
			fmt.Printf("skipping job %d because it is not in the account list\n", doc.RequestId)
			continue label
		}
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
			log.Debugf("tx success, Hash: \n\t[%s]", txResponse.TxHash)
		} else {
			log.Debugf("tx failed: %s", txResponse.RawLog)

			// sequence mismatch 에러인 경우 sequence 동기화 시도
			if txResponse.Code == 32 { // sequence mismatch error code
				log.Debugf("sequence mismatch error, attempting to sync sequence number")
				if err := d.transactionManager.SyncSequenceNumber(); err != nil {
					log.Debugf("failed to sync sequence: %v", err)
				} else {
					log.Debugf("sequence number synchronized")
				}
			}
		}

		// 성공/실패 관계없이 sequence number 증가
		// 블록체인에서 sequence는 한번 사용되면 소모됨
		d.transactionManager.IncrementSequenceNumber()
	}
}
