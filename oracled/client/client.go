package client

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	cmtypes "github.com/tendermint/tendermint/types"

	"github.com/GPTx-global/guru/oracled/types"
)

type Client struct {
	config      *Config
	rpcClient   *http.HTTP
	isConnected bool

	// Client가 생성하고 소유하는 채널 (이벤트 생성자)
	eventCh chan coretypes.ResultEvent

	// 새로운 컴포넌트들
	txBuilder *TxBuilder
	// Scheduler로부터 받는 채널 (Oracle 데이터 소비자)
	oracleResultCh <-chan types.OracleData
}

func NewClient() *Client {
	config := LoadConfig()
	return &Client{
		config:      config,
		rpcClient:   nil,
		isConnected: false,
		// Client가 이벤트를 생성하므로 EventCh 생성
		eventCh: make(chan coretypes.ResultEvent, 64),
		// Oracle 데이터는 Scheduler에서 받으므로 nil로 초기화
		oracleResultCh: nil,
	}
}

// EventCh의 읽기 전용 채널을 반환 (Scheduler가 사용)
func (c *Client) GetEventChannel() <-chan coretypes.ResultEvent {
	return c.eventCh
}

// Scheduler의 OracleDataCh를 설정 (읽기 전용으로 받음)
func (c *Client) SetOracleResultChannel(ch <-chan types.OracleData) {
	c.oracleResultCh = ch
}

func (c *Client) Connect() error {
	if c.isConnected {
		return nil
	}

	var err error
	c.rpcClient, err = http.New(c.config.rpcEndpoint, "/websocket")
	if err != nil {
		return errors.New("failed to create rpc client: " + err.Error())
	}
	fmt.Println("rpc client created")

	err = c.rpcClient.Start()
	if err != nil {
		return errors.New("failed to start rpc client: " + err.Error())
	}
	fmt.Println("rpc client started")

	// TxBuilder 초기화
	c.txBuilder, err = NewTxBuilder(c.config)
	if err != nil {
		return fmt.Errorf("failed to create tx builder: %w", err)
	}
	fmt.Println("tx builder created")

	c.isConnected = true

	return nil
}

func (c *Client) Disconnect() error {
	if !c.isConnected {
		return nil
	}

	c.rpcClient.Stop()
	c.rpcClient = nil
	c.isConnected = false

	return nil
}

func (c *Client) Monitor(ctx context.Context) error {
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

	fmt.Println("Started monitoring blockchain events...")
	fmt.Println("- Subscribing to block events (job requests)")
	fmt.Println("- Subscribing to oracle complete events")

	for {
		select {
		case event := <-blockCh:
			c.handleBlockEvent(event)
		case event := <-oracleCh:
			c.handleOracleCompleteEvent(event)
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *Client) handleBlockEvent(event coretypes.ResultEvent) {
	if event.Data != nil {
		if block, ok := event.Data.(cmtypes.EventDataNewBlock); ok {
			fmt.Printf("새로운 블록 생성됨: 높이=%d, 해시=%X\n",
				block.Block.Height,
				block.Block.Hash())
		}

		// 작업 요청 이벤트 확인 후 scheduler로 전달
		if c.isJobRequestEvent(event) {
			select {
			case c.eventCh <- event:
				fmt.Printf("Job request event forwarded to scheduler\n")
			default:
				fmt.Printf("Event channel full, dropping job request event\n")
			}
		}
	}
}

func (c *Client) handleOracleCompleteEvent(event coretypes.ResultEvent) {
	if event.Data != nil {
		fmt.Printf("Oracle complete event detected\n")

		// Oracle 완료 이벤트를 scheduler로 전달
		select {
		case c.eventCh <- event:
			fmt.Printf("Oracle complete event forwarded to scheduler\n")
		default:
			fmt.Printf("Event channel full, dropping oracle complete event\n")
		}
	}
}

// 작업 요청 이벤트인지 확인
func (c *Client) isJobRequestEvent(event coretypes.ResultEvent) bool {
	// 기본적인 타입 확인만 수행 (파싱은 하지 않음)
	if event.Data == nil {
		return false
	}

	// 간단한 이벤트 타입 확인
	if strings.Contains(event.Query, "oracle_request") ||
		strings.Contains(event.Query, "NewBlock") { // 임시로 모든 블록 이벤트 포함
		return true
	}

	return false
}

func (c *Client) PrepareOracle(ctx context.Context) {
	fmt.Println("Started oracle transaction processor...")

	for {
		select {
		case oracleResult := <-c.oracleResultCh:
			// Oracle 데이터를 받아서 트랜잭션 생성 및 전송
			err := c.processOracleResult(ctx, oracleResult)
			if err != nil {
				fmt.Printf("Failed to process oracle result: %v\n", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) processOracleResult(ctx context.Context, oracleResult types.OracleData) error {
	fmt.Printf("Processing oracle result: %s\n", oracleResult.RequestID)

	// 트랜잭션 생성 (단일 OracleData를 배열로 감싸서 전달)
	oracleResults := []types.OracleData{oracleResult}
	txBytes, err := c.txBuilder.BuildOracleTx(ctx, oracleResults)
	if err != nil {
		return fmt.Errorf("failed to build oracle tx: %w", err)
	}

	// 트랜잭션 전송
	resp, err := c.txBuilder.BroadcastTx(ctx, txBytes)
	if err != nil {
		return fmt.Errorf("failed to broadcast oracle tx: %w", err)
	}

	fmt.Printf("Oracle transaction sent successfully: %s\n", resp.TxHash)
	return nil
}
