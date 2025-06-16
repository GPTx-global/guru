package subscribe

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
	"github.com/GPTx-global/guru/oracle/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

var (
	registerMsg = "/guru.oracle.v1.MsgRegisterOracleRequestDoc"
	updateMsg   = "/guru.oracle.v1.MsgUpdateOracleRequestDoc"
	completeMsg = "complete_oracle_data_set"
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
	res, err := client.OracleRequestDocs(sm.ctx, &oracletypes.QueryOracleRequestDocsRequest{})
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

	completeQuery := fmt.Sprintf("tm.event='NewBlock' AND %s.request_id EXISTS", completeMsg)
	ch, err = client.Subscribe(sm.ctx, completeMsg, completeQuery, sm.channelSize)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", completeMsg, err)
	}
	sm.subscriptions[completeMsg] = ch

	log.Debugf("end setting subscribe")

	return nil
}

// Subscribe listens for events from all subscribed channels and converts them to jobs
func (sm *SubscribeManager) Subscribe() []*types.Job {
	sm.subscriptionsLock.RLock()
	defer sm.subscriptionsLock.RUnlock()

	select {
	case event := <-sm.subscriptions[registerMsg]:
		if !sm.filterAccount(event, oracletypes.EventTypeRegisterOracleRequestDoc) {
			return nil
		}
		// TODO: daemon account에 할달되었는지 확인 + 할당된 경우 어느 account가 어느 작업을 수행하는지 확인
		// id, _ := strconv.ParseUint(event.Events[oracletypes.EventTypeRegisterOracleRequestDoc+"."+oracletypes.AttributeKeyRequestId][0], 10, 64)
		// nonce, _ := strconv.ParseUint(event.Events[oracletypes.EventTypeRegisterOracleRequestDoc+"."+oracletypes.AttributeKeyNonce][0], 10, 64)
		// fmt.Printf("[MONITOR-REGISTER] ID: %5d, Nonce: %5d\n", id, nonce)

		return types.MakeJobs(event)
	case event := <-sm.subscriptions[updateMsg]:
		if !sm.filterAccount(event, oracletypes.EventTypeUpdateOracleRequestDoc) {
			return nil
		}
		return types.MakeJobs(event)
	case event := <-sm.subscriptions[completeMsg]:
		return types.MakeJobs(event)
	case <-sm.ctx.Done():
		return nil
	}
}

func (sm *SubscribeManager) filterAccount(event coretypes.ResultEvent, prefix string) bool {
	accounts := event.Events[prefix+"."+oracletypes.AttributeKeyAccountList][0]

	return strings.Contains(accounts, config.Config.Address())
}
