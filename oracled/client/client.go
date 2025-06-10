package client

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	fmt.Printf("[ START ] NewClient\n")

	c := &Client{
		config:      LoadConfig(),
		rpcClient:   nil,
		isConnected: false,
		eventCh:     make(chan coretypes.ResultEvent, 64),
		resultCh:    nil,
	}

	fmt.Printf("[  END  ] NewClient: SUCCESS - eventCh buffer size=64\n")
	return c
}

func (c *Client) Start(ctx context.Context) error {
	fmt.Printf("[ START ] Start\n")

	if err := c.connect(); err != nil {
		c.disconnect()
		fmt.Printf("[  END  ] Start: ERROR - %v\n", err)
		return errors.New("failed to connect to rpc: " + err.Error())
	}

	go c.monitor(ctx)
	go c.serveOracle(ctx)

	fmt.Printf("[  END  ] Start: SUCCESS\n")
	return nil
}

func (c *Client) connect() error {
	fmt.Printf("[ START ] connect - RPC: %s\n", c.config.rpcEndpoint)

	if c.isConnected {
		fmt.Printf("[  END  ] connect: SUCCESS - already connected\n")
		return nil
	}

	var err error
	if c.rpcClient, err = http.New(c.config.rpcEndpoint, "/websocket"); err != nil {
		fmt.Printf("[  END  ] connect: ERROR - failed to create rpc client: %v\n", err)
		return errors.New("failed to create rpc client: " + err.Error())
	}

	if err = c.rpcClient.Start(); err != nil {
		fmt.Printf("[  END  ] connect: ERROR - failed to start rpc client: %v\n", err)
		return errors.New("failed to start rpc client: " + err.Error())
	}

	if c.txBuilder, err = NewTxBuilder(c.config, c.rpcClient); err != nil {
		fmt.Printf("[  END  ] connect: ERROR - failed to create tx builder: %v\n", err)
		return fmt.Errorf("failed to create tx builder: %w", err)
	}

	c.isConnected = true
	fmt.Printf("[  END  ] connect: SUCCESS - connected to %s\n", c.config.rpcEndpoint)
	return nil
}

func (c *Client) disconnect() error {
	fmt.Printf("[ START ] disconnect\n")

	if !c.isConnected {
		fmt.Printf("[  END  ] disconnect: SUCCESS - already disconnected\n")
		return nil
	}

	c.rpcClient.Stop()
	c.rpcClient = nil
	c.isConnected = false

	fmt.Printf("[  END  ] disconnect: SUCCESS\n")
	return nil
}

func (c *Client) monitor(ctx context.Context) error {
	fmt.Printf("[ START ] monitor\n")

	if !c.isConnected {
		err := errors.New("not connected to rpc")
		fmt.Printf("[  END  ] monitor: ERROR - %v\n", err)
		return err
	}

	// Oracle 작업 등록 트랜잭션
	queryRegisterOracle := "tm.event='Tx' AND message.action='/guru.oracle.v1.MsgRegisterOracleRequestDoc'"
	registerCh, err := c.rpcClient.Subscribe(ctx, "register_oracle_subscribe", queryRegisterOracle)
	if err != nil {
		fmt.Printf("[  END  ] monitor: ERROR - failed to subscribe register events: %v\n", err)
		return errors.New("failed to subscribe to oracle register events: " + err.Error())
	}

	// Oracle 작업 수정 트랜잭션
	queryUpdateOracle := "tm.event='Tx' AND message.action='/guru.oracle.v1.MsgUpdateOracleRequestDoc'"
	updateCh, err := c.rpcClient.Subscribe(ctx, "update_oracle_subscribe", queryUpdateOracle)
	if err != nil {
		fmt.Printf("[  END  ] monitor: ERROR - failed to subscribe update events: %v\n", err)
		return errors.New("failed to subscribe to oracle update events: " + err.Error())
	}

	// Oracle 사용 완료 이벤트
	queryCompleteOracle := fmt.Sprintf("tm.event='NewBlock' AND %s EXISTS", "complete_oracle_data_set.request_id")
	completeCh, err := c.rpcClient.Subscribe(ctx, "complete_oracle_subscribe", queryCompleteOracle)
	if err != nil {
		fmt.Printf("[  END  ] monitor: ERROR - failed to subscribe complete events: %v\n", err)
		return errors.New("failed to subscribe to oracle complete events: " + err.Error())
	}

	for {
		select {
		case event := <-registerCh:
			fmt.Printf("[ EVENT ] monitor: Received register oracle event\n")
			c.checkEvent("register_oracle_request_doc", event)
		case event := <-updateCh:
			fmt.Printf("[ EVENT ] monitor: Received update oracle event\n")
			c.checkEvent("", event)
		case event := <-completeCh:
			fmt.Printf("[ EVENT ] monitor: Received complete oracle event\n")
			c.checkEvent("", event)
		case <-ctx.Done():
			fmt.Printf("[  END  ] monitor: SUCCESS - context cancelled\n")
			return nil
		}
	}
}

func (c *Client) checkEvent(prefix string, event coretypes.ResultEvent) {
	fmt.Printf("[ START ] checkEvent - prefix: %s\n", prefix)

	if event.Data == nil {
		fmt.Printf("[  END  ] checkEvent: SKIP - event data is nil\n")
		return
	}

	if prefix == "register_oracle_request_doc" {
		requestIDs := event.Events["register_oracle_request_doc.account_list"][0]
		if !strings.Contains(requestIDs, c.txBuilder.clientCtx.GetFromAddress().String()) {
			fmt.Printf("[  END  ] checkEvent: SKIP - requestID not found\n")
			return
		}
	}

	c.eventCh <- event

	fmt.Printf("[  END  ] checkEvent: SUCCESS\n")
}

func (c *Client) serveOracle(ctx context.Context) {
	fmt.Printf("[ START ] serveOracle\n")

	for {
		select {
		case oracleResult := <-c.resultCh:
			fmt.Printf("[RESULT ] serveOracle: Received oracle result for request ID: %d\n", oracleResult.RequestID)
			go c.processTransaction(ctx, oracleResult)
		case <-ctx.Done():
			fmt.Printf("[  END  ] serveOracle: SUCCESS - context cancelled\n")
			return
		}
	}
}

func (c *Client) processTransaction(ctx context.Context, oracleResult types.OracleData) {
	fmt.Printf("[ START ] processTransaction - RequestID: %d, Nonce: %d\n",
		oracleResult.RequestID, oracleResult.Nonce)

	txBytes, err := c.txBuilder.BuildOracleTx(ctx, oracleResult)
	if err != nil {
		fmt.Printf("[  END  ] processTransaction: ERROR - failed to build tx: %v\n", err)
		return
	}

	resp, err := c.txBuilder.BroadcastTx(ctx, txBytes)
	if err != nil {
		fmt.Printf("[  END  ] processTransaction: ERROR - failed to broadcast tx: %v\n", err)
		return
	}
	c.txBuilder.incSequence()
	fmt.Printf("[SUCCESS] Hash: %s\n", resp.TxHash)
	fmt.Printf("[  END  ] processTransaction: SUCCESS\n")
	fmt.Printf("=================================================================================================\n")
}

func (c *Client) GetEventChannel() <-chan coretypes.ResultEvent {
	return c.eventCh
}

func (c *Client) SetResultChannel(ch <-chan types.OracleData) {
	c.resultCh = ch
}
