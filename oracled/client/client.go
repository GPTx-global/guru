package client

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/encoding"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/GPTx-global/guru/oracled/retry"
	"github.com/GPTx-global/guru/oracled/types"
)

type Client struct {
	config      *Config
	rpcClient   *http.HTTP
	isConnected bool
	txBuilder   *TxBuilder
	jobCh       chan *types.Job
	resultCh    <-chan types.OracleData

	// Job 관리
	activeJobs    map[uint64]*types.Job
	activeJobsMux sync.Mutex
	txDecoder     sdk.TxDecoder
	cdc           codec.Codec

	// 기본적인 회로 차단기만 유지
	connectCB   *retry.CircuitBreaker
	reconnectMu sync.Mutex

	// 구독 관리 (중요한 로직이므로 유지)
	subscriptions   map[string]<-chan coretypes.ResultEvent
	subscriptionsMu sync.RWMutex
	lastEventTime   map[string]time.Time
	isSubscribed    bool
}

func NewClient() *Client {
	fmt.Printf("[ START ] NewClient\n")

	encodingConfig := encoding.MakeConfig(app.ModuleBasics)
	c := &Client{
		config:        LoadConfig(),
		rpcClient:     nil,
		isConnected:   false,
		jobCh:         make(chan *types.Job, 256),
		resultCh:      nil,
		activeJobs:    make(map[uint64]*types.Job),
		activeJobsMux: sync.Mutex{},
		txDecoder:     encodingConfig.TxConfig.TxDecoder(),
		cdc:           encodingConfig.Codec,
		connectCB:     retry.NewCircuitBreaker(5, 5*time.Minute),
	}

	fmt.Printf("[  END  ] NewClient: SUCCESS - jobCh buffer size=256\n")
	return c
}

func (c *Client) Start(ctx context.Context) error {
	fmt.Printf("[ START ] Client.Start\n")

	// 구독 관리 초기화
	c.subscriptions = make(map[string]<-chan coretypes.ResultEvent)
	c.lastEventTime = make(map[string]time.Time)
	c.isSubscribed = false

	// 단순한 연결 시도
	if err := c.connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Oracle 문서 조회 및 Job 생성
	tempC := oracletypes.NewQueryClient(c.txBuilder.clientCtx)
	res, err := tempC.OracleRequestDocs(ctx, &oracletypes.QueryOracleRequestDocsRequest{})
	if err != nil {
		fmt.Printf("[  END  ] Start: ERROR - %v\n", err)
		return fmt.Errorf("failed to query oracle request docs: %w", err)
	}

	fmt.Printf("------------------------\n")
	for _, doc := range res.OracleRequestDocs {
		fmt.Printf("ID: %d\n", doc.RequestId)
		fmt.Printf("Nonce: %d\n", doc.Nonce)
		fmt.Printf("Status: %s\n", doc.Status)
		fmt.Printf("Endpoints: %v\n", doc.Endpoints)
		fmt.Printf("Period: %d\n", doc.Period)
		fmt.Printf("AccountList: %v\n", doc.AccountList)
		fmt.Printf("Description: %s\n", doc.Description)
		fmt.Printf("Name: %s\n", doc.Name)
		fmt.Printf("------------------------\n")
		job := &types.Job{
			ID:     doc.RequestId,
			URL:    doc.Endpoints[0].Url,
			Path:   doc.Endpoints[0].ParseRule,
			Nonce:  doc.Nonce,
			Delay:  time.Duration(doc.Period) * time.Second,
			Status: doc.Status.String(),
		}
		c.activeJobs[doc.RequestId] = job
		c.jobCh <- job
	}

	// 백그라운드 프로세스 시작
	go c.simpleMonitor(ctx)
	go c.serveOracle(ctx)

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
	fmt.Printf("[  END  ] connect: SUCCESS - connected to %s\n", c.config.rpcEndpoint)
	return nil
}

// 단순한 모니터 - 복잡한 재연결 로직 제거하되 이벤트 구독은 유지
func (c *Client) simpleMonitor(ctx context.Context) {
	fmt.Printf("[ START ] simpleMonitor\n")

	// 이벤트 구독 (중요한 로직이므로 유지)
	if err := c.subscribeToEventsWithMonitoring(ctx); err != nil {
		fmt.Printf("[  ERROR] simpleMonitor: Failed to subscribe to events: %v\n", err)
	}

	<-ctx.Done()
	fmt.Printf("[  END  ] simpleMonitor: context cancelled\n")
}

func (c *Client) subscribeToEventsWithMonitoring(ctx context.Context) error {
	fmt.Printf("[ START ] subscribeToEventsWithMonitoring\n")

	if !c.isConnected {
		return fmt.Errorf("not connected to rpc")
	}

	// 기존 구독이 있으면 정리
	c.cleanupSubscriptions()

	// 모든 중요한 이벤트 구독 (원래대로 복원)
	subscriptions := map[string]string{
		"register_oracle": "tm.event='Tx' AND message.action='/guru.oracle.v1.MsgRegisterOracleRequestDoc'",
		"update_oracle":   "tm.event='Tx' AND message.action='/guru.oracle.v1.MsgUpdateOracleRequestDoc'",
		"complete_oracle": fmt.Sprintf("tm.event='NewBlock' AND %s EXISTS", "complete_oracle_data_set.request_id"),
	}

	channels := make(map[string]<-chan coretypes.ResultEvent)

	// 모든 구독을 순차적으로 생성
	for name, query := range subscriptions {
		fmt.Printf("[  INFO ] subscribeToEventsWithMonitoring: Subscribing to %s\n", name)

		ch, err := c.rpcClient.Subscribe(ctx, name+"_subscribe", query, 64)
		if err != nil {
			fmt.Printf("[  ERROR] subscribeToEventsWithMonitoring: Failed to subscribe %s: %v\n", name, err)
			// 에러 처리 단순화: 실패해도 계속 진행
			continue
		}

		channels[name] = ch
		c.lastEventTime[name] = time.Now()
		fmt.Printf("[  INFO ] subscribeToEventsWithMonitoring: Successfully subscribed to %s\n", name)
	}

	// 구독 상태 업데이트
	c.subscriptionsMu.Lock()
	c.subscriptions = channels
	c.isSubscribed = true
	c.subscriptionsMu.Unlock()

	// 이벤트 루프 시작 (중요한 로직이므로 유지)
	go c.enhancedEventLoop(ctx)

	fmt.Printf("[  END  ] subscribeToEventsWithMonitoring: SUCCESS\n")
	return nil
}

func (c *Client) enhancedEventLoop(ctx context.Context) {
	fmt.Printf("[ START ] enhancedEventLoop\n")

	// 모든 구독 채널을 하나의 select에서 처리 (중요한 로직)
	for {
		c.subscriptionsMu.RLock()
		registerCh := c.subscriptions["register_oracle"]
		updateCh := c.subscriptions["update_oracle"]
		completeCh := c.subscriptions["complete_oracle"]
		isSubscribed := c.isSubscribed
		c.subscriptionsMu.RUnlock()

		if !isSubscribed {
			fmt.Printf("[  WARN ] enhancedEventLoop: Not subscribed, exiting\n")
			return
		}

		select {
		case event, ok := <-registerCh:
			if !ok {
				fmt.Printf("[  WARN ] enhancedEventLoop: registerCh closed\n")
				return
			}
			fmt.Printf("[ EVENT ] enhancedEventLoop: Received register oracle event\n")
			c.updateLastEventTime("register_oracle")
			c.checkEvent("register_oracle_request_doc", event)

		case event, ok := <-updateCh:
			if !ok {
				fmt.Printf("[  WARN ] enhancedEventLoop: updateCh closed\n")
				return
			}
			fmt.Printf("[ EVENT ] enhancedEventLoop: Received update oracle event\n")
			c.updateLastEventTime("update_oracle")
			c.checkEvent("", event)

		case event, ok := <-completeCh:
			if !ok {
				fmt.Printf("[  WARN ] enhancedEventLoop: completeCh closed\n")
				return
			}
			fmt.Printf("[ EVENT ] enhancedEventLoop: Received complete oracle event\n")
			c.updateLastEventTime("complete_oracle")
			c.checkEvent("", event)

		case <-ctx.Done():
			fmt.Printf("[  END  ] enhancedEventLoop: SUCCESS - context cancelled\n")
			return

		case <-time.After(30 * time.Second):
			// 주기적 헬스 체크 (30초마다)
			fmt.Printf("[ HEALTH] enhancedEventLoop: Periodic health check\n")
			continue
		}
	}
}

func (c *Client) updateLastEventTime(subscriptionName string) {
	c.subscriptionsMu.Lock()
	defer c.subscriptionsMu.Unlock()
	c.lastEventTime[subscriptionName] = time.Now()
}

func (c *Client) cleanupSubscriptions() {
	c.subscriptionsMu.Lock()
	defer c.subscriptionsMu.Unlock()

	c.subscriptions = make(map[string]<-chan coretypes.ResultEvent)
	c.lastEventTime = make(map[string]time.Time)
	c.isSubscribed = false
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

	// 이벤트를 job으로 변환
	job, err := c.eventToJob(event)
	if err != nil {
		fmt.Printf("[  WARN ] checkEvent: Failed to convert event to job: %v\n", err)
		return
	}

	if job == nil {
		fmt.Printf("[  INFO ] checkEvent: Event converted to nil job (not an error)\n")
		return
	}

	// Job 채널에 전송 (논블로킹)
	select {
	case c.jobCh <- job:
		fmt.Printf("[  END  ] checkEvent: SUCCESS - job sent to scheduler\n")
	default:
		fmt.Printf("[  WARN ] checkEvent: jobCh buffer full, dropping job\n")
	}
}

// eventToJob: 이벤트를 Job으로 변환 (원래 로직 복원)
func (c *Client) eventToJob(event coretypes.ResultEvent) (*types.Job, error) {
	fmt.Printf("[ START ] eventToJob\n")
	fmt.Printf("[  INFO ] eventToJob: Event data type: %T\n", event.Data)

	switch eventData := event.Data.(type) {
	case tmtypes.EventDataTx:
		fmt.Printf("[  INFO ] eventToJob: Processing Tx event\n")
		// 트랜잭션 바이트 데이터를 디코딩
		tx, err := c.txDecoder(eventData.Tx)
		if err != nil {
			fmt.Printf("[  END  ] eventToJob: ERROR - failed to decode transaction: %v\n", err)
			return nil, fmt.Errorf("failed to decode transaction: %w", err)
		}

		// 트랜잭션 내의 모든 메시지를 확인
		msgs := tx.GetMsgs()
		fmt.Printf("[  INFO ] eventToJob: Transaction has %d messages\n", len(msgs))

		for i, msg := range msgs {
			fmt.Printf("[  INFO ] eventToJob: Processing message %d/%d - Type: %T\n", i+1, len(msgs), msg)

			switch oracleMsg := msg.(type) {
			case *oracletypes.MsgRegisterOracleRequestDoc:
				fmt.Printf("[  INFO ] eventToJob: Found MsgRegisterOracleRequestDoc\n")

				// Register Oracle Request 메시지 처리 - 첫 번째 등록
				if len(event.Events["register_oracle_request_doc.request_id"]) == 0 {
					fmt.Printf("[  ERROR] eventToJob: request_id not found in event\n")
					return nil, fmt.Errorf("request_id not found in event")
				}

				reqID, err := strconv.ParseUint(event.Events["register_oracle_request_doc.request_id"][0], 10, 64)
				if err != nil {
					fmt.Printf("[  ERROR] eventToJob: Failed to parse request_id: %v\n", err)
					return nil, fmt.Errorf("failed to parse request_id: %w", err)
				}

				fmt.Printf("[  INFO ] eventToJob: Parsed request_id: %d\n", reqID)

				c.activeJobsMux.Lock()
				// 이미 존재하는 job인지 확인
				if _, exists := c.activeJobs[reqID]; exists {
					c.activeJobsMux.Unlock()
					fmt.Printf("[  WARN ] eventToJob: Job already exists: %d\n", reqID)
					return nil, fmt.Errorf("job already exists: %d", reqID)
				}

				if len(oracleMsg.RequestDoc.Endpoints) == 0 {
					c.activeJobsMux.Unlock()
					fmt.Printf("[  ERROR] eventToJob: No endpoints in RequestDoc\n")
					return nil, fmt.Errorf("no endpoints in RequestDoc")
				}

				job := &types.Job{
					ID:     reqID,
					URL:    oracleMsg.RequestDoc.Endpoints[0].Url,
					Path:   oracleMsg.RequestDoc.Endpoints[0].ParseRule,
					Nonce:  oracleMsg.RequestDoc.Nonce + 1,
					Delay:  time.Duration(oracleMsg.RequestDoc.Period) * time.Second,
					Status: event.Events["register_oracle_request_doc.status"][0],
				}

				// activeJobs에 추가
				c.activeJobs[reqID] = job
				c.activeJobsMux.Unlock()

				fmt.Printf("[SUCCESS] eventToJob: Registered new job - ID: %d, URL: %s, Path: %s, Delay: %v\n",
					job.ID, job.URL, job.Path, job.Delay)

				// 첫 번째 실행을 위해 job 반환
				fmt.Printf("[  END  ] eventToJob: SUCCESS - new job created\n")
				return job, nil

			case *oracletypes.MsgUpdateOracleRequestDoc:
				fmt.Printf("[  INFO ] eventToJob: Found MsgUpdateOracleRequestDoc\n")
				// Update Oracle Request 메시지 처리
				reqID := oracleMsg.RequestDoc.RequestId

				c.activeJobsMux.Lock()
				if existingJob, exists := c.activeJobs[reqID]; exists {
					// 기존 job 업데이트
					existingJob.URL = oracleMsg.RequestDoc.Endpoints[0].Url
					existingJob.Path = oracleMsg.RequestDoc.Endpoints[0].ParseRule
					existingJob.Delay = time.Duration(oracleMsg.RequestDoc.Period) * time.Second
					c.activeJobsMux.Unlock()

					fmt.Printf("[SUCCESS] eventToJob: Updated existing job - ID: %d, URL: %s, Path: %s\n",
						existingJob.ID, existingJob.URL, existingJob.Path)
					fmt.Printf("[  END  ] eventToJob: SUCCESS - job updated\n")
					return existingJob, nil
				}
				c.activeJobsMux.Unlock()
				fmt.Printf("[  WARN ] eventToJob: Job not found for update: %d\n", reqID)
				return nil, fmt.Errorf("job not found for update: %d", reqID)

			default:
				fmt.Printf("[  INFO ] eventToJob: Ignoring message type: %T\n", msg)
			}
		}

	case tmtypes.EventDataNewBlock:
		fmt.Printf("[  INFO ] eventToJob: Processing NewBlock event\n")

		if len(event.Events["complete_oracle_data_set.request_id"]) == 0 {
			fmt.Printf("[  ERROR] eventToJob: complete_oracle_data_set.request_id not found in NewBlock event\n")
			return nil, fmt.Errorf("complete_oracle_data_set.request_id not found in NewBlock event")
		}

		reqID, err := strconv.ParseUint(event.Events["complete_oracle_data_set.request_id"][0], 10, 64)
		if err != nil {
			fmt.Printf("[  ERROR] eventToJob: Failed to parse request ID from NewBlock: %v\n", err)
			return nil, fmt.Errorf("failed to parse request ID: %w", err)
		}

		fmt.Printf("[  INFO ] eventToJob: NewBlock event for request_id: %d\n", reqID)

		c.activeJobsMux.Lock()
		existingJob, exists := c.activeJobs[reqID]
		if !exists {
			c.activeJobsMux.Unlock()
			fmt.Printf("[  WARN ] eventToJob: Job not found in activeJobs for NewBlock: %d\n", reqID)
			return nil, fmt.Errorf("job not found in activeJobs: %d", reqID)
		}

		// 기존 job의 nonce 증가
		oldNonce := existingJob.Nonce
		existingJob.Nonce++
		fmt.Printf("[SUCCESS] eventToJob: Incremented nonce for job ID=%d: %d -> %d\n",
			reqID, oldNonce, existingJob.Nonce)
		c.activeJobsMux.Unlock()

		// 업데이트된 job 반환
		fmt.Printf("[  END  ] eventToJob: SUCCESS - nonce incremented\n")
		return existingJob, nil

	default:
		fmt.Printf("[  WARN ] eventToJob: Unsupported event data type: %T\n", event.Data)
		return nil, fmt.Errorf("unsupported event data type: %T", event.Data)
	}

	fmt.Printf("[  WARN ] eventToJob: No matching message found in transaction\n")
	return nil, nil
}

func (c *Client) serveOracle(ctx context.Context) {
	fmt.Printf("[ START ] serveOracle\n")

	for {
		select {
		case oracleResult := <-c.resultCh:
			fmt.Printf("[ RESULT] serveOracle: Received oracle result for request ID: %d\n", oracleResult.RequestID)
			if err := c.processTransaction(ctx, oracleResult); err != nil {
				fmt.Printf("[  WARN ] serveOracle: Failed to process transaction: %v\n", err)
			}

			// 간단한 트랜잭션 처리
			// go func(result types.OracleData) {
			// 	if err := c.processTransaction(ctx, result); err != nil {
			// 		fmt.Printf("[  WARN ] serveOracle: Failed to process transaction: %v\n", err)
			// 		// 기본적인 재시도 한 번만
			// 		time.Sleep(2 * time.Second)
			// 		if err := c.processTransaction(ctx, result); err != nil {
			// 			fmt.Printf("[  ERROR] serveOracle: Transaction failed after retry: %v\n", err)
			// 		}
			// 	}
			// }(oracleResult)

		case <-ctx.Done():
			fmt.Printf("[  END  ] serveOracle: SUCCESS - context cancelled\n")
			return
		}
	}
}

func (c *Client) processTransaction(ctx context.Context, oracleResult types.OracleData) error {
	fmt.Printf("[ START ] processTransaction - RequestID: %d, Nonce: %d\n",
		oracleResult.RequestID, oracleResult.Nonce)

	// 연결 상태 확인
	if !c.isConnected {
		if err := c.connect(); err != nil {
			return fmt.Errorf("reconnection failed: %w", err)
		}
	}

	// TxBuilder 상태 확인
	if c.txBuilder == nil {
		var err error
		if c.txBuilder, err = NewTxBuilder(c.config, c.rpcClient); err != nil {
			return fmt.Errorf("failed to recreate TxBuilder: %w", err)
		}
	}

	txBytes, err := c.txBuilder.BuildOracleTx(ctx, oracleResult)
	if err != nil {
		return fmt.Errorf("failed to build tx: %w", err)
	}
	c.txBuilder.incSequence()

	resp, err := c.txBuilder.BroadcastTx(ctx, txBytes)
	if err != nil {
		return fmt.Errorf("failed to broadcast tx: %w", err)
	}

	fmt.Printf("[SUCCESS] Hash: %s\n", resp.TxHash)
	fmt.Printf("[  END  ] processTransaction: SUCCESS\n")
	fmt.Printf("=================================================================================================\n")

	return nil
}

func (c *Client) GetJobChannel() <-chan *types.Job {
	return c.jobCh
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
