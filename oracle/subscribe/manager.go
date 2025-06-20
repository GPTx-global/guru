package subscribe

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

var (
	registerMsg = "/guru.oracle.v1.MsgRegisterOracleRequestDoc"
	updateMsg   = "/guru.oracle.v1.MsgUpdateOracleRequestDoc"
)

type SubscribeManager struct {
	subscriptions     map[string]<-chan coretypes.ResultEvent
	subscriptionsLock sync.RWMutex
	channelSize       int
	ctx               context.Context
}

// NewSubscribeManager creates a new subscription manager for blockchain events
func NewSubscribeManager(ctx context.Context) *SubscribeManager {
	return &SubscribeManager{
		subscriptions:     make(map[string]<-chan coretypes.ResultEvent),
		subscriptionsLock: sync.RWMutex{},
		channelSize:       2 << 10,
		ctx:               ctx,
	}
}

// LoadRegisterRequest queries and returns all existing oracle request documents from the blockchain
func (sm *SubscribeManager) LoadRegisterRequest(clientCtx client.Context) ([]*oracletypes.OracleRequestDoc, error) {
	log.Debugf("start loading register request")
	client := oracletypes.NewQueryClient(clientCtx)
	res, err := client.OracleRequestDocs(sm.ctx, &oracletypes.QueryOracleRequestDocsRequest{
		Status: oracletypes.RequestStatus_REQUEST_STATUS_ENABLED,
	})
	if err != nil {
		log.Errorf("failed to load register request: %v", err)
		return nil, fmt.Errorf("failed to load register request: %w", err)
	}
	log.Debugf("end loading register request, %d", len(res.OracleRequestDocs))

	return res.OracleRequestDocs, nil
}

// SetSubscribe establishes subscriptions to oracle-related blockchain events
func (sm *SubscribeManager) SetSubscribe(client *http.HTTP) error {
	log.Debugf("start setting subscribe")
	sm.subscriptionsLock.Lock()
	defer sm.subscriptionsLock.Unlock()

	registerQuery := fmt.Sprintf("tm.event='Tx' AND message.action='%s'", registerMsg)
	ch, err := client.Subscribe(sm.ctx, registerMsg, registerQuery, sm.channelSize)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", registerMsg, err)
	}
	sm.subscriptions[registerMsg] = ch

	updateQuery := fmt.Sprintf("tm.event='Tx' AND message.action='%s'", updateMsg)
	ch, err = client.Subscribe(sm.ctx, updateMsg, updateQuery, sm.channelSize)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", updateMsg, err)
	}
	sm.subscriptions[updateMsg] = ch

	completeQuery := fmt.Sprintf("tm.event='NewBlock' AND %s.%s EXISTS", oracletypes.EventTypeCompleteOracleDataSet, oracletypes.AttributeKeyRequestId)
	ch, err = client.Subscribe(sm.ctx, oracletypes.EventTypeCompleteOracleDataSet, completeQuery, sm.channelSize)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", oracletypes.EventTypeCompleteOracleDataSet, err)
	}
	sm.subscriptions[oracletypes.EventTypeCompleteOracleDataSet] = ch

	// minGasPriceQuery := fmt.Sprintf("tm.event='NewBlock' AND %s.%s EXISTS", feemarkettypes.EventTypeChangeMinGasPrice, feemarkettypes.AttributeKeyMinGasPrice)
	// ch, err = client.Subscribe(sm.ctx, feemarkettypes.EventTypeChangeMinGasPrice, minGasPriceQuery, sm.channelSize)
	// if err != nil {
	// 	return fmt.Errorf("failed to subscribe to %s: %w", feemarkettypes.EventTypeChangeMinGasPrice, err)
	// }
	// sm.subscriptions[feemarkettypes.EventTypeChangeMinGasPrice] = ch

	log.Debugf("end setting subscribe")

	return nil
}

// Subscribe listens for events from all subscribed channels and converts them to jobs
func (sm *SubscribeManager) Subscribe() *coretypes.ResultEvent {
	sm.subscriptionsLock.RLock()
	defer sm.subscriptionsLock.RUnlock()

	select {
	case event := <-sm.subscriptions[registerMsg]:
		if !sm.filterAccount(event, oracletypes.EventTypeRegisterOracleRequestDoc) {
			return nil
		}
		return &event
	case event := <-sm.subscriptions[updateMsg]:
		return &event
	case event := <-sm.subscriptions[oracletypes.EventTypeCompleteOracleDataSet]:
		return &event
	// case event := <-sm.subscriptions[feemarkettypes.EventTypeChangeMinGasPrice]:
	// 	return &event
	case <-sm.ctx.Done():
		return nil
	default:
		return nil
	}
}

func (sm *SubscribeManager) filterAccount(event coretypes.ResultEvent, prefix string) bool {
	accounts := event.Events[prefix+"."+oracletypes.AttributeKeyAccountList][0]

	return strings.Contains(accounts, config.Address().String())
}
