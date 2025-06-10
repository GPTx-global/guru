package client

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/GPTx-global/guru/oracled/retry"
	"github.com/GPTx-global/guru/oracled/types"
)

type Client struct {
	config      *Config
	rpcClient   *http.HTTP
	isConnected bool
	txBuilder   *TxBuilder
	eventCh     chan coretypes.ResultEvent
	resultCh    <-chan types.OracleData

	// 에러 처리를 위한 추가 필드
	connectCB   *retry.CircuitBreaker
	reconnectMu sync.Mutex
	lastConnect time.Time
}

func NewClient() *Client {
	fmt.Printf("[ START ] NewClient\n")

	c := &Client{
		config:      LoadConfig(),
		rpcClient:   nil,
		isConnected: false,
		eventCh:     make(chan coretypes.ResultEvent, 64),
		resultCh:    nil,
		connectCB:   retry.NewCircuitBreaker(5, 5*time.Minute),
	}

	fmt.Printf("[  END  ] NewClient: SUCCESS - eventCh buffer size=64\n")
	return c
}

func (c *Client) Start(ctx context.Context) error {
	fmt.Printf("[ START ] Start\n")

	// 연결 재시도 로직
	err := retry.Do(ctx, retry.NetworkRetryConfig(),
		func() error {
			return c.connectCB.Execute(func() error {
				return c.connect()
			})
		},
		retry.DefaultIsRetryable,
	)

	if err != nil {
		c.disconnect()
		fmt.Printf("[  END  ] Start: ERROR - %v\n", err)
		return fmt.Errorf("failed to connect to rpc after retries: %w", err)
	}

	go c.monitor(ctx)
	go c.serveOracle(ctx)
	go c.connectionWatchdog(ctx) // 연결 감시 고루틴 추가

	fmt.Printf("[  END  ] Start: SUCCESS\n")
	return nil
}

func (c *Client) connect() error {
	fmt.Printf("[ START ] connect - RPC: %s\n", c.config.rpcEndpoint)

	c.reconnectMu.Lock()
	defer c.reconnectMu.Unlock()

	if c.isConnected {
		fmt.Printf("[  END  ] connect: SUCCESS - already connected\n")
		return nil
	}

	var err error
	if c.rpcClient, err = http.New(c.config.rpcEndpoint, "/websocket"); err != nil {
		fmt.Printf("[  END  ] connect: ERROR - failed to create rpc client: %v\n", err)
		return fmt.Errorf("failed to create rpc client: %w", err)
	}

	if err = c.rpcClient.Start(); err != nil {
		fmt.Printf("[  END  ] connect: ERROR - failed to start rpc client: %v\n", err)
		return fmt.Errorf("failed to start rpc client: %w", err)
	}

	if c.txBuilder, err = NewTxBuilder(c.config, c.rpcClient); err != nil {
		fmt.Printf("[  END  ] connect: ERROR - failed to create tx builder: %v\n", err)
		return fmt.Errorf("failed to create tx builder: %w", err)
	}

	c.isConnected = true
	c.lastConnect = time.Now()
	fmt.Printf("[  END  ] connect: SUCCESS - connected to %s\n", c.config.rpcEndpoint)
	return nil
}

func (c *Client) disconnect() error {
	fmt.Printf("[ START ] disconnect\n")

	c.reconnectMu.Lock()
	defer c.reconnectMu.Unlock()

	if !c.isConnected {
		fmt.Printf("[  END  ] disconnect: SUCCESS - already disconnected\n")
		return nil
	}

	if c.rpcClient != nil {
		c.rpcClient.Stop()
		c.rpcClient = nil
	}
	c.isConnected = false

	fmt.Printf("[  END  ] disconnect: SUCCESS\n")
	return nil
}

func (c *Client) monitor(ctx context.Context) error {
	fmt.Printf("[ START ] monitor\n")

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[  END  ] monitor: SUCCESS - context cancelled\n")
			return nil
		default:
		}

		// 연결 상태 확인 및 재연결
		if !c.isConnected {
			fmt.Printf("[  WARN ] monitor: RPC not connected, attempting reconnection\n")

			err := retry.Do(ctx, retry.NetworkRetryConfig(),
				func() error {
					return c.connectCB.Execute(func() error {
						return c.connect()
					})
				},
				retry.DefaultIsRetryable,
			)

			if err != nil {
				fmt.Printf("[  WARN ] monitor: Failed to reconnect: %v\n", err)
				time.Sleep(10 * time.Second) // 재시도 전 대기
				continue
			}
		}

		// 이벤트 구독 시도
		err := c.subscribeToEvents(ctx)
		if err != nil {
			fmt.Printf("[  WARN ] monitor: Event subscription failed: %v\n", err)
			c.disconnect() // 연결 문제로 인한 구독 실패 시 재연결 유도
			time.Sleep(5 * time.Second)
			continue
		}

		break // 성공적으로 구독했으면 루프 종료
	}

	fmt.Printf("[  END  ] monitor: SUCCESS - monitoring started\n")
	return nil
}

func (c *Client) subscribeToEvents(ctx context.Context) error {
	fmt.Printf("[ START ] subscribeToEvents\n")

	if !c.isConnected {
		return fmt.Errorf("not connected to rpc")
	}

	// Oracle 작업 등록 트랜잭션
	queryRegisterOracle := "tm.event='Tx' AND message.action='/guru.oracle.v1.MsgRegisterOracleRequestDoc'"
	registerCh, err := c.rpcClient.Subscribe(ctx, "register_oracle_subscribe", queryRegisterOracle)
	if err != nil {
		fmt.Printf("[  END  ] subscribeToEvents: ERROR - failed to subscribe register events: %v\n", err)
		return fmt.Errorf("failed to subscribe to oracle register events: %w", err)
	}

	// Oracle 작업 수정 트랜잭션
	queryUpdateOracle := "tm.event='Tx' AND message.action='/guru.oracle.v1.MsgUpdateOracleRequestDoc'"
	updateCh, err := c.rpcClient.Subscribe(ctx, "update_oracle_subscribe", queryUpdateOracle)
	if err != nil {
		fmt.Printf("[  END  ] subscribeToEvents: ERROR - failed to subscribe update events: %v\n", err)
		return fmt.Errorf("failed to subscribe to oracle update events: %w", err)
	}

	// Oracle 사용 완료 이벤트
	queryCompleteOracle := fmt.Sprintf("tm.event='NewBlock' AND %s EXISTS", "complete_oracle_data_set.request_id")
	completeCh, err := c.rpcClient.Subscribe(ctx, "complete_oracle_subscribe", queryCompleteOracle)
	if err != nil {
		fmt.Printf("[  END  ] subscribeToEvents: ERROR - failed to subscribe complete events: %v\n", err)
		return fmt.Errorf("failed to subscribe to oracle complete events: %w", err)
	}

	// 이벤트 리스닝 시작
	go c.eventLoop(ctx, registerCh, updateCh, completeCh)

	fmt.Printf("[  END  ] subscribeToEvents: SUCCESS\n")
	return nil
}

func (c *Client) eventLoop(ctx context.Context, registerCh, updateCh, completeCh <-chan coretypes.ResultEvent) {
	fmt.Printf("[ START ] eventLoop\n")

	for {
		select {
		case event, ok := <-registerCh:
			if !ok {
				fmt.Printf("[  WARN ] eventLoop: registerCh closed, restarting monitor\n")
				go c.monitor(ctx)
				return
			}
			fmt.Printf("[ EVENT ] monitor: Received register oracle event\n")
			c.checkEvent("register_oracle_request_doc", event)

		case event, ok := <-updateCh:
			if !ok {
				fmt.Printf("[  WARN ] eventLoop: updateCh closed, restarting monitor\n")
				go c.monitor(ctx)
				return
			}
			fmt.Printf("[ EVENT ] monitor: Received update oracle event\n")
			c.checkEvent("", event)

		case event, ok := <-completeCh:
			if !ok {
				fmt.Printf("[  WARN ] eventLoop: completeCh closed, restarting monitor\n")
				go c.monitor(ctx)
				return
			}
			fmt.Printf("[ EVENT ] monitor: Received complete oracle event\n")
			c.checkEvent("", event)

		case <-ctx.Done():
			fmt.Printf("[  END  ] eventLoop: SUCCESS - context cancelled\n")
			return
		}
	}
}

func (c *Client) connectionWatchdog(ctx context.Context) {
	fmt.Printf("[ START ] connectionWatchdog\n")

	ticker := time.NewTicker(30 * time.Second) // 30초마다 연결 상태 체크
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if c.isConnected && time.Since(c.lastConnect) > 5*time.Minute {
				// 5분 이상 연결이 지속되었는데 문제가 없다면 건강한 상태
				fmt.Printf("[ HEALTH] connectionWatchdog: Connection healthy\n")
			}

			// RPC 헬스 체크 (간단한 ping)
			if c.isConnected && c.rpcClient != nil {
				_, err := c.rpcClient.Status(ctx)
				if err != nil {
					fmt.Printf("[  WARN ] connectionWatchdog: RPC health check failed: %v\n", err)
					c.disconnect()
				}
			}

		case <-ctx.Done():
			fmt.Printf("[  END  ] connectionWatchdog: SUCCESS - context cancelled\n")
			return
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
		// 계정 목록에서 현재 계정이 포함되어 있는지 확인
		if accountList, exists := event.Events["register_oracle_request_doc.account_list"]; exists && len(accountList) > 0 {
			if !strings.Contains(accountList[0], c.txBuilder.clientCtx.GetFromAddress().String()) {
				fmt.Printf("[  END  ] checkEvent: SKIP - account not in list\n")
				return
			}
		} else {
			fmt.Printf("[  END  ] checkEvent: SKIP - account_list not found\n")
			return
		}
	}

	// 이벤트 채널에 전송 (논블로킹)
	select {
	case c.eventCh <- event:
		fmt.Printf("[  END  ] checkEvent: SUCCESS\n")
	default:
		fmt.Printf("[  WARN ] checkEvent: eventCh buffer full, dropping event\n")
	}
}

func (c *Client) serveOracle(ctx context.Context) {
	fmt.Printf("[ START ] serveOracle\n")

	for {
		select {
		case oracleResult := <-c.resultCh:
			fmt.Printf("[ RESULT] serveOracle: Received oracle result for request ID: %d\n", oracleResult.RequestID)

			// 트랜잭션 처리를 별도 고루틴에서 처리 (논블로킹)
			go func(result types.OracleData) {
				// 트랜잭션 처리 재시도 로직
				err := retry.Do(ctx, retry.TransactionRetryConfig(),
					func() error {
						return c.processTransactionWithRecovery(ctx, result)
					},
					retry.TransactionIsRetryable,
				)

				if err != nil {
					fmt.Printf("[  WARN ] serveOracle: Failed to process transaction after retries: %v\n", err)
				}
			}(oracleResult)

		case <-ctx.Done():
			fmt.Printf("[  END  ] serveOracle: SUCCESS - context cancelled\n")
			return
		}
	}
}

func (c *Client) processTransactionWithRecovery(ctx context.Context, oracleResult types.OracleData) error {
	fmt.Printf("[ START ] processTransactionWithRecovery - RequestID: %d, Nonce: %d\n",
		oracleResult.RequestID, oracleResult.Nonce)

	// 연결 상태 확인
	if !c.isConnected {
		fmt.Printf("[  WARN ] processTransactionWithRecovery: Not connected, attempting reconnection\n")
		if err := c.connect(); err != nil {
			fmt.Printf("[  END  ] processTransactionWithRecovery: ERROR - reconnection failed: %v\n", err)
			return fmt.Errorf("reconnection failed: %w", err)
		}
	}

	// TxBuilder 상태 확인 및 복구
	if c.txBuilder == nil {
		fmt.Printf("[  WARN ] processTransactionWithRecovery: TxBuilder is nil, recreating\n")
		var err error
		if c.txBuilder, err = NewTxBuilder(c.config, c.rpcClient); err != nil {
			fmt.Printf("[  END  ] processTransactionWithRecovery: ERROR - failed to recreate TxBuilder: %v\n", err)
			return fmt.Errorf("failed to recreate TxBuilder: %w", err)
		}
	}

	return c.processTransaction(ctx, oracleResult)
}

func (c *Client) processTransaction(ctx context.Context, oracleResult types.OracleData) error {
	fmt.Printf("[ START ] processTransaction - RequestID: %d, Nonce: %d\n",
		oracleResult.RequestID, oracleResult.Nonce)

	txBytes, err := c.txBuilder.BuildOracleTx(ctx, oracleResult)
	if err != nil {
		fmt.Printf("[  END  ] processTransaction: ERROR - failed to build tx: %v\n", err)
		return fmt.Errorf("failed to build tx: %w", err)
	}

	resp, err := c.txBuilder.BroadcastTx(ctx, txBytes)
	if err != nil {
		fmt.Printf("[  END  ] processTransaction: ERROR - failed to broadcast tx: %v\n", err)
		return fmt.Errorf("failed to broadcast tx: %w", err)
	}

	c.txBuilder.incSequence()
	fmt.Printf("[SUCCESS] Hash: %s\n", resp.TxHash)
	fmt.Printf("[  END  ] processTransaction: SUCCESS\n")
	fmt.Printf("=================================================================================================\n")

	return nil
}

func (c *Client) GetEventChannel() <-chan coretypes.ResultEvent {
	return c.eventCh
}

func (c *Client) SetResultChannel(ch <-chan types.OracleData) {
	c.resultCh = ch
}

// IsConnected 연결 상태 확인 (헬스 체크용)
func (c *Client) IsConnected() bool {
	return c.isConnected && c.rpcClient != nil
}

// GetHealthCheckFunc 헬스 체크 함수 반환
func (c *Client) GetHealthCheckFunc() func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if !c.IsConnected() {
			return fmt.Errorf("rpc client not connected")
		}

		// 간단한 RPC 호출로 연결 상태 확인
		_, err := c.rpcClient.Status(ctx)
		if err != nil {
			return fmt.Errorf("rpc status check failed: %w", err)
		}

		return nil
	}
}
