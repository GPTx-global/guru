package monitor

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
	"github.com/GPTx-global/guru/oracle/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

type Monitor struct {
	clientContext     client.Context
	backgroundContext context.Context
	registerChannel   <-chan coretypes.ResultEvent
	updateChannel     <-chan coretypes.ResultEvent
	completeChannel   <-chan coretypes.ResultEvent
}

func New(clientContext client.Context, backgroundContext context.Context) *Monitor {
	return &Monitor{
		clientContext:     clientContext,
		backgroundContext: backgroundContext,
		registerChannel:   nil,
		updateChannel:     nil,
		completeChannel:   nil,
	}
}

func (m *Monitor) Start() {
	log.Debugf("monitor manager starting")
	m.initChannels()
}

func (m *Monitor) Stop() {
	m.registerChannel = nil
	m.updateChannel = nil
	m.completeChannel = nil
	log.Debugf("monitor manager stopped")
}

func (m *Monitor) initChannels() {
	var err error
	registerQuery := "tm.event='Tx' AND message.action='/guru.oracle.v1.MsgRegisterOracleRequestDoc'"
	m.registerChannel, err = m.clientContext.Client.Subscribe(m.backgroundContext, "register", registerQuery, config.ChannelSize())
	if err != nil {
		panic(fmt.Errorf("failed to subscribe to register: %w", err))
	}

	updateQuery := "tm.event='Tx' AND message.action='/guru.oracle.v1.MsgUpdateOracleRequestDoc'"
	m.updateChannel, err = m.clientContext.Client.Subscribe(m.backgroundContext, "update", updateQuery, config.ChannelSize())
	if err != nil {
		panic(fmt.Errorf("failed to subscribe to update: %w", err))
	}

	completeQuery := fmt.Sprintf("tm.event='NewBlock' AND %s.%s EXISTS", oracletypes.EventTypeCompleteOracleDataSet, oracletypes.AttributeKeyRequestId)
	m.completeChannel, err = m.clientContext.Client.Subscribe(m.backgroundContext, "complete", completeQuery, config.ChannelSize())
	if err != nil {
		panic(fmt.Errorf("failed to subscribe to complete: %w", err))
	}

	log.Debugf("init channels successfully")
}

func (m *Monitor) LoadRequestDocs() ([]*oracletypes.OracleRequestDoc, error) {
	client := oracletypes.NewQueryClient(m.clientContext)
	res, err := client.OracleRequestDocs(m.backgroundContext, &oracletypes.QueryOracleRequestDocsRequest{Status: oracletypes.RequestStatus_REQUEST_STATUS_ENABLED})
	if err != nil {
		log.Errorf("failed to load register request: %v", err)
		return nil, fmt.Errorf("failed to load register request: %w", err)
	}

	return res.OracleRequestDocs, nil
}

func (m *Monitor) Subscribe() any {
	select {
	case event := <-m.registerChannel:
		id, err := strconv.ParseUint(event.Events[types.RegisterID][0], 10, 64)
		if err != nil {
			log.Debugf("failed to parse id: %v", err)
			return nil
		}

		accounts := event.Events[types.RegisterAccountList][0]
		if !strings.Contains(accounts, config.Address().String()) {
			log.Debugf("register event not for me: %d", id)
			return nil
		}

		queryRes, err := oracletypes.NewQueryClient(m.clientContext).OracleRequestDoc(m.backgroundContext, &oracletypes.QueryOracleRequestDocRequest{RequestId: id})
		if err != nil {
			log.Debugf("failed to query oracle request doc: %v", err)
			return nil
		}

		return queryRes.RequestDoc

	case event := <-m.updateChannel:
		id, err := strconv.ParseUint(event.Events[types.UpdateID][0], 10, 64)
		if err != nil {
			log.Debugf("failed to parse id: %v", err)
			return nil
		}

		queryRes, err := oracletypes.NewQueryClient(m.clientContext).OracleRequestDoc(m.backgroundContext, &oracletypes.QueryOracleRequestDocRequest{RequestId: id})
		if err != nil {
			log.Debugf("failed to query oracle request doc: %v", err)
			return nil
		}

		return queryRes.RequestDoc

	case event := <-m.completeChannel:

		return event
	}
}
