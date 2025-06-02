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

	if c.txBuilder, err = NewTxBuilder(c.config); err != nil {
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

	// 1. 블록 이벤트 구독 (작업 요청 tx 포함)
	queryBlock := "tm.event='NewBlock'"
	blockCh, err := c.rpcClient.Subscribe(ctx, "block_subscribe", queryBlock)
	if err != nil {
		return errors.New("failed to subscribe to block events: " + err.Error())
	}

	// 2. Oracle 완료 이벤트 구독
	queryOracleComplete := "tm.event='Tx' AND complete_oracle_data_set EXISTS"
	oracleCh, err := c.rpcClient.Subscribe(ctx, "oracle_subscribe", queryOracleComplete)
	if err != nil {
		return errors.New("failed to subscribe to oracle complete events: " + err.Error())
	}

	for {
		select {
		case event := <-blockCh:
			c.checkEvent(event)
		case event := <-oracleCh:
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

	// TODO: 이벤트 타입 확인하는 로직 추가

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
	fmt.Printf("Processing oracle result: %s\n", oracleResult.RequestID)

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
}

func (c *Client) GetEventChannel() <-chan coretypes.ResultEvent {
	return c.eventCh
}

func (c *Client) SetResultChannel(ch <-chan types.OracleData) {
	c.resultCh = ch
}
