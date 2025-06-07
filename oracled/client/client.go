package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/GPTx-global/guru/oracled/types"
)

type Client struct {
	config      *Config
	rpcClient   *http.HTTP
	isConnected bool
	txBuilder   *TxBuilder
	eventCh     chan coretypes.ResultEvent
	resultCh    <-chan types.OracleData
}

func NewClient() *Client {
	return &Client{
		config:      LoadConfig(),
		rpcClient:   nil,
		isConnected: false,
		eventCh:     make(chan coretypes.ResultEvent, 64),
		resultCh:    nil,
	}
}

func (c *Client) Start(ctx context.Context) error {
	if err := c.connect(); err != nil {
		c.disconnect()
		return errors.New("failed to connect to rpc: " + err.Error())
	}

	go c.monitor(ctx)
	go c.serveOracle(ctx)

	return nil
}

func (c *Client) connect() error {
	if c.isConnected {
		return nil
	}

	var err error
	if c.rpcClient, err = http.New(c.config.rpcEndpoint, "/websocket"); err != nil {
		return errors.New("failed to create rpc client: " + err.Error())
	}

	if err = c.rpcClient.Start(); err != nil {
		return errors.New("failed to start rpc client: " + err.Error())
	}

	if c.txBuilder, err = NewTxBuilder(c.config, c.rpcClient); err != nil {
		return fmt.Errorf("failed to create tx builder: %w", err)
	}

	c.isConnected = true

	return nil
}

func (c *Client) disconnect() error {
	if !c.isConnected {
		return nil
	}

	c.rpcClient.Stop()
	c.rpcClient = nil
	c.isConnected = false

	return nil
}

func (c *Client) monitor(ctx context.Context) error {
	if !c.isConnected {
		return errors.New("not connected to rpc")
	}

	// Oracle 작업 등록 트랜잭션
	queryRegisterOracle := "tm.event='Tx' AND message.action='/guru.oracle.v1.MsgRegisterOracleRequestDoc'"
	registerCh, err := c.rpcClient.Subscribe(ctx, "register_oracle_subscribe", queryRegisterOracle)
	if err != nil {
		return errors.New("failed to subscribe to oracle register events: " + err.Error())
	}

	// Oracle 작업 수정 트랜잭션
	queryUpdateOracle := "tm.event='Tx' AND message.action='/guru.oracle.v1.MsgUpdateOracleRequestDoc'"
	updateCh, err := c.rpcClient.Subscribe(ctx, "update_oracle_subscribe", queryUpdateOracle)
	if err != nil {
		return errors.New("failed to subscribe to oracle update events: " + err.Error())
	}

	// Oracle 사용 완료 이벤트
	queryCompleteOracle := fmt.Sprintf("tm.event='NewBlock' AND %s EXISTS", "complete_oracle_data_set.request_id")
	// queryCompleteOracle := fmt.Sprintf("tm.event='NewBlock' AND %s.OracleId EXISTS", "alpha")
	completeCh, err := c.rpcClient.Subscribe(ctx, "complete_oracle_subscribe", queryCompleteOracle)
	if err != nil {
		return errors.New("failed to subscribe to oracle complete events: " + err.Error())
	}

	for {
		select {
		case event := <-registerCh:
			c.checkEvent(event)
		case event := <-updateCh:
			c.checkEvent(event)
		case event := <-completeCh:
			c.checkEvent(event)
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *Client) checkEvent(event coretypes.ResultEvent) {
	if event.Data == nil {
		return
	}

	/*
		TODO: 이벤트 검사하는 로직 추가
	*/

	c.eventCh <- event
}

func (c *Client) serveOracle(ctx context.Context) {
	for {
		select {
		case oracleResult := <-c.resultCh:
			go c.processTransaction(ctx, oracleResult)
		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) processTransaction(ctx context.Context, oracleResult types.OracleData) {
	fmt.Printf("Processing oracle result: %d\n", oracleResult.RequestID)

	txBytes, err := c.txBuilder.BuildOracleTx(ctx, oracleResult)
	if err != nil {
		fmt.Printf("Failed to build oracle tx: %v\n", err)
		return
	}

	resp, err := c.txBuilder.BroadcastTx(ctx, txBytes)
	if err != nil {
		fmt.Printf("Failed to broadcast oracle tx: %v\n", err)
		return
	}

	fmt.Printf("Oracle transaction sent successfully: %s\n", resp.TxHash)
	c.txBuilder.incSequence()
}

func (c *Client) GetEventChannel() <-chan coretypes.ResultEvent {
	return c.eventCh
}

func (c *Client) SetResultChannel(ch <-chan types.OracleData) {
	c.resultCh = ch
}
