package daemon

import (
	"context"
	"fmt"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/encoding"
	"github.com/GPTx-global/guru/oracle/subscribe"
	"github.com/GPTx-global/guru/oracle/types"
	"github.com/GPTx-global/guru/oracle/tx"
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

	jobChannel chan *types.Job

	ctx context.Context
}

func New(ctx context.Context) (*Daemon, error) {
	d := new(Daemon)
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
	d.jobChannel = make(chan *types.Job, 64)
	d.jobManager = woker.NewJobManager()
	return d, nil
}

func (d *Daemon) Start() error {
	d.jobManager.Start(d.ctx)

	err := d.client.Start()
	if err != nil {
		return fmt.Errorf("failed to start client: %w", err)
	}

	docs, err := d.subscribeManager.LoadRegisterRequest(d.clientCtx)
	if err != nil {
		return fmt.Errorf("failed to load register request: %w", err)
	}
	for _, doc := range docs {
		if job := types.MakeJob(doc); job != nil {
			d.ProcessJob(job)
		}
	}

	err = d.subscribeManager.SetSubscribe(d.client)
	if err != nil {
		return fmt.Errorf("failed to set subscribe: %w", err)
	}

	go d.Monitor()

	return nil
}

func (d *Daemon) Stop() {
	d.jobManager.Stop()
	d.client.Stop()
	d.ctx.Done()
}

func (d *Daemon) Monitor() {
	for {
		if job := d.subscribeManager.Subscribe(); job != nil {
			d.ProcessJob(job)
		}
	}
}

func (d *Daemon) ProcessJob(job *types.Job) {
	d.jobManager.SubmitJob(job)
}

func (d *Daemon) Serve() error {
	return nil
}
