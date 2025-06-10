package scheduler

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/encoding"
	"github.com/GPTx-global/guru/oracled/retry"
	"github.com/GPTx-global/guru/oracled/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type Scheduler struct {
	eventCh       <-chan coretypes.ResultEvent
	activeJobs    map[uint64]*types.Job
	activeJobsMux sync.Mutex
	resultCh      chan types.OracleData
	txDecoder     sdk.TxDecoder
	cdc           codec.Codec

	// 에러 처리를 위한 추가 필드
	failedEvents    []coretypes.ResultEvent
	failedEventsMux sync.Mutex
	lastHealthCheck time.Time

	// 성능 최적화를 위한 워커 풀
	workerPool *JobWorkerPool
}

func NewScheduler() *Scheduler {
	fmt.Printf("[ START ] NewScheduler\n")

	encodingConfig := encoding.MakeConfig(app.ModuleBasics)
	s := &Scheduler{
		eventCh:         nil,
		activeJobs:      make(map[uint64]*types.Job),
		activeJobsMux:   sync.Mutex{},
		resultCh:        make(chan types.OracleData, 256), // 버퍼 크기 증가
		txDecoder:       encodingConfig.TxConfig.TxDecoder(),
		cdc:             encodingConfig.Codec,
		failedEvents:    make([]coretypes.ResultEvent, 0),
		lastHealthCheck: time.Now(),
		workerPool:      nil, // Start()에서 초기화
	}

	fmt.Printf("[  END  ] NewScheduler: SUCCESS - resultCh buffer size=256\n")
	return s
}

func (s *Scheduler) Start(ctx context.Context) {
	fmt.Printf("[ START ] Start\n")

	// 워커 풀 초기화 및 시작 (CPU 코어 수 * 4개의 워커)
	maxWorkers := runtime.NumCPU() * 4 // 동시 처리 가능한 최대 job 수 제한
	s.workerPool = NewJobWorkerPool(maxWorkers, s)
	s.workerPool.Start(ctx)

	go s.eventProcessor(ctx)
	go s.failedEventRetryProcessor(ctx) // 실패한 이벤트 재처리 고루틴
	go s.healthMonitor(ctx)             // 헬스 모니터링 고루틴

	fmt.Printf("[  END  ] Start: SUCCESS - WorkerPool with %d workers\n", maxWorkers)
}

func (s *Scheduler) eventProcessor(ctx context.Context) {
	fmt.Printf("[ START ] eventProcessor\n")

	for {
		select {
		case event := <-s.eventCh:
			fmt.Printf("[ EVENT ] eventProcessor: Received event - Type: %T\n", event.Data)
			fmt.Printf("[ EVENT ] eventProcessor: Event query: %s\n", event.Query)
			fmt.Printf("[ EVENT ] eventProcessor: Event events: %+v\n", event.Events)

			// 이벤트 처리를 재시도 로직으로 래핑
			err := retry.Do(ctx, retry.DefaultRetryConfig(),
				func() error {
					job, err := s.eventToJob(event)
					if err != nil {
						fmt.Printf("[  ERROR] eventProcessor: Failed to convert event to job: %v\n", err)
						return fmt.Errorf("failed to convert event to job: %w", err)
					}

					if job == nil {
						fmt.Printf("[  INFO ] eventProcessor: Event converted to nil job (not an error)\n")
						return nil // job이 nil인 경우는 에러가 아님
					}

					fmt.Printf("[  INFO ] eventProcessor: Successfully converted event to job - ID: %d, Nonce: %d\n", job.ID, job.Nonce)

					// 워커 풀에 job 제출 (고루틴 무제한 생성 방지)
					fmt.Printf("[  INFO ] eventProcessor: Submitting job to worker pool\n")
					if !s.workerPool.SubmitJob(job) {
						fmt.Printf("[  WARN ] eventProcessor: Worker pool queue full\n")
						return fmt.Errorf("worker pool queue full")
					}
					fmt.Printf("[  INFO ] eventProcessor: Successfully submitted job to worker pool\n")
					return nil
				},
				func(err error) bool {
					// 일시적인 파싱 에러나 시스템 에러만 재시도
					return retry.DefaultIsRetryable(err)
				},
			)

			if err != nil {
				fmt.Printf("[  WARN ] eventProcessor: Failed to process event after retries: %v\n", err)
				s.addFailedEvent(event) // 실패한 이벤트를 별도로 저장
			} else {
				fmt.Printf("[SUCCESS] eventProcessor: Event processed successfully\n")
			}

		case <-ctx.Done():
			fmt.Printf("[  END  ] eventProcessor: SUCCESS - context cancelled\n")
			return
		}
	}
}

func (s *Scheduler) processJobWithRetry(ctx context.Context, job *types.Job) {
	fmt.Printf("[ START ] processJobWithRetry - ID: %d, Nonce: %d, Status: %s\n",
		job.ID, job.Nonce, job.Status)
	fmt.Printf("[  INFO ] processJobWithRetry: Job details - URL: %s, Path: %s, Delay: %v\n",
		job.URL, job.Path, job.Delay)

	// Job 처리를 재시도 로직으로 래핑
	err := retry.Do(ctx, retry.DefaultRetryConfig(),
		func() error {
			fmt.Printf("[  INFO ] processJobWithRetry: Attempting to process job %d\n", job.ID)
			return s.processJob(ctx, job)
		},
		func(err error) bool {
			// 네트워크 에러나 일시적 에러만 재시도
			return retry.DefaultIsRetryable(err)
		},
	)

	if err != nil {
		fmt.Printf("[  WARN ] processJobWithRetry: Failed to process job %d after retries: %v\n",
			job.ID, err)
	} else {
		fmt.Printf("[SUCCESS] processJobWithRetry: Job %d processed successfully\n", job.ID)
	}

	fmt.Printf("[  END  ] processJobWithRetry\n")
}

func (s *Scheduler) failedEventRetryProcessor(ctx context.Context) {
	fmt.Printf("[ START ] failedEventRetryProcessor\n")

	ticker := time.NewTicker(2 * time.Minute) // 2분마다 실패한 이벤트 재처리
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.retryFailedEvents(ctx)
		case <-ctx.Done():
			fmt.Printf("[  END  ] failedEventRetryProcessor: SUCCESS - context cancelled\n")
			return
		}
	}
}

func (s *Scheduler) retryFailedEvents(ctx context.Context) {
	s.failedEventsMux.Lock()
	events := make([]coretypes.ResultEvent, len(s.failedEvents))
	copy(events, s.failedEvents)
	s.failedEvents = s.failedEvents[:0] // 리스트 클리어
	s.failedEventsMux.Unlock()

	if len(events) == 0 {
		return
	}

	fmt.Printf("[ RETRY ] retryFailedEvents: Retrying %d failed events\n", len(events))

	for _, event := range events {
		select {
		case <-ctx.Done():
			return
		default:
		}

		job, err := s.eventToJob(event)
		if err != nil {
			fmt.Printf("[  WARN ] retryFailedEvents: Still failing to process event: %v\n", err)
			s.addFailedEvent(event) // 다시 실패 목록에 추가
			continue
		}

		if job != nil {
			// 워커 풀에 job 제출
			if !s.workerPool.SubmitJob(job) {
				fmt.Printf("[  WARN ] retryFailedEvents: Worker pool queue full, re-adding to failed events\n")
				s.addFailedEvent(event)
			}
		}
	}
}

func (s *Scheduler) healthMonitor(ctx context.Context) {
	fmt.Printf("[ START ] healthMonitor\n")

	ticker := time.NewTicker(1 * time.Minute) // 1분마다 헬스 체크
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performHealthCheck()
		case <-ctx.Done():
			fmt.Printf("[  END  ] healthMonitor: SUCCESS - context cancelled\n")
			return
		}
	}
}

func (s *Scheduler) performHealthCheck() {
	s.activeJobsMux.Lock()
	activeJobsCount := len(s.activeJobs)
	s.activeJobsMux.Unlock()

	s.failedEventsMux.Lock()
	failedEventsCount := len(s.failedEvents)
	s.failedEventsMux.Unlock()

	resultChLen := len(s.resultCh)
	resultChCap := cap(s.resultCh)

	fmt.Printf("[ HEALTH] Scheduler status - ActiveJobs: %d, FailedEvents: %d, ResultCh: %d/%d\n",
		activeJobsCount, failedEventsCount, resultChLen, resultChCap)

	// 결과 채널이 거의 가득 찬 경우 경고
	if float64(resultChLen)/float64(resultChCap) > 0.8 {
		fmt.Printf("[  WARN ] healthMonitor: Result channel is %d%% full\n",
			(resultChLen*100)/resultChCap)
	}

	// 실패한 이벤트가 너무 많은 경우 경고
	if failedEventsCount > 100 {
		fmt.Printf("[  WARN ] healthMonitor: Too many failed events: %d\n", failedEventsCount)
	}

	s.lastHealthCheck = time.Now()
}

func (s *Scheduler) addFailedEvent(event coretypes.ResultEvent) {
	s.failedEventsMux.Lock()
	defer s.failedEventsMux.Unlock()

	// 실패한 이벤트 목록이 너무 커지지 않도록 제한
	if len(s.failedEvents) < 1000 {
		s.failedEvents = append(s.failedEvents, event)
	} else {
		fmt.Printf("[  WARN ] addFailedEvent: Failed events queue is full, dropping oldest events\n")
		// FIFO로 가장 오래된 이벤트 제거하고 새 이벤트 추가
		copy(s.failedEvents, s.failedEvents[1:])
		s.failedEvents[len(s.failedEvents)-1] = event
	}
}

func (s *Scheduler) eventToJob(event coretypes.ResultEvent) (*types.Job, error) {
	fmt.Printf("[ START ] eventToJob\n")
	fmt.Printf("[  INFO ] eventToJob: Event data type: %T\n", event.Data)

	switch eventData := event.Data.(type) {
	case tmtypes.EventDataTx:
		fmt.Printf("[  INFO ] eventToJob: Processing Tx event\n")
		// 트랜잭션 바이트 데이터를 디코딩
		tx, err := s.txDecoder(eventData.Tx)
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

				s.activeJobsMux.Lock()
				// 이미 존재하는 job인지 확인
				if _, exists := s.activeJobs[reqID]; exists {
					s.activeJobsMux.Unlock()
					fmt.Printf("[  WARN ] eventToJob: Job already exists: %d\n", reqID)
					return nil, fmt.Errorf("job already exists: %d", reqID)
				}

				if len(oracleMsg.RequestDoc.Endpoints) == 0 {
					s.activeJobsMux.Unlock()
					fmt.Printf("[  ERROR] eventToJob: No endpoints in RequestDoc\n")
					return nil, fmt.Errorf("no endpoints in RequestDoc")
				}

				job := &types.Job{
					ID:     reqID,
					URL:    oracleMsg.RequestDoc.Endpoints[0].Url,
					Path:   oracleMsg.RequestDoc.Endpoints[0].ParseRule,
					Nonce:  1,
					Delay:  time.Duration(oracleMsg.RequestDoc.Period) * time.Second,
					Status: event.Events["register_oracle_request_doc.status"][0],
				}

				// activeJobs에 추가
				s.activeJobs[reqID] = job
				s.activeJobsMux.Unlock()

				fmt.Printf("[SUCCESS] eventToJob: Registered new job - ID: %d, URL: %s, Path: %s, Delay: %v\n",
					job.ID, job.URL, job.Path, job.Delay)

				// 첫 번째 실행을 위해 job 반환
				fmt.Printf("[  END  ] eventToJob: SUCCESS - new job created\n")
				return job, nil

			case *oracletypes.MsgUpdateOracleRequestDoc:
				fmt.Printf("[  INFO ] eventToJob: Found MsgUpdateOracleRequestDoc\n")
				// Update Oracle Request 메시지 처리
				reqID := oracleMsg.RequestDoc.RequestId

				s.activeJobsMux.Lock()
				if existingJob, exists := s.activeJobs[reqID]; exists {
					// 기존 job 업데이트
					existingJob.URL = oracleMsg.RequestDoc.Endpoints[0].Url
					existingJob.Path = oracleMsg.RequestDoc.Endpoints[0].ParseRule
					existingJob.Delay = time.Duration(oracleMsg.RequestDoc.Period) * time.Second
					s.activeJobsMux.Unlock()

					fmt.Printf("[SUCCESS] eventToJob: Updated existing job - ID: %d, URL: %s, Path: %s\n",
						existingJob.ID, existingJob.URL, existingJob.Path)
					fmt.Printf("[  END  ] eventToJob: SUCCESS - job updated\n")
					return existingJob, nil
				}
				s.activeJobsMux.Unlock()
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

		s.activeJobsMux.Lock()
		existingJob, exists := s.activeJobs[reqID]
		if !exists {
			s.activeJobsMux.Unlock()
			fmt.Printf("[  WARN ] eventToJob: Job not found in activeJobs for NewBlock: %d\n", reqID)
			return nil, fmt.Errorf("job not found in activeJobs: %d", reqID)
		}

		// 기존 job의 nonce 증가
		oldNonce := existingJob.Nonce
		existingJob.Nonce++
		fmt.Printf("[SUCCESS] eventToJob: Incremented nonce for job ID=%d: %d -> %d\n",
			reqID, oldNonce, existingJob.Nonce)
		s.activeJobsMux.Unlock()

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

func (s *Scheduler) processJob(ctx context.Context, job *types.Job) error {
	fmt.Printf("[ START ] processJob - ID: %d, Nonce: %d, Status: %s\n", job.ID, job.Nonce, job.Status)
	fmt.Printf("[  INFO ] processJob: Creating executor for job\n")

	executor := NewExecutor(ctx)

	fmt.Printf("[  INFO ] processJob: Executing job %d\n", job.ID)
	oracleData, err := executor.ExecuteJob(job)
	if err != nil {
		fmt.Printf("[  END  ] processJob: ERROR - failed to execute job %d: %v\n", job.ID, err)
		return fmt.Errorf("failed to execute job %d: %w", job.ID, err)
	}

	fmt.Printf("[  INFO ] processJob: Job executed successfully, sending to result channel\n")
	fmt.Printf("[  INFO ] processJob: Oracle data - RequestID: %d, Data: %s, Nonce: %d\n",
		oracleData.RequestID, oracleData.Data, oracleData.Nonce)

	// 결과 채널에 전송 (논블로킹)
	select {
	case s.resultCh <- *oracleData:
		fmt.Printf("[SUCCESS] processJob: Oracle data sent to result channel - RequestID: %d\n", oracleData.RequestID)
		fmt.Printf("[  END  ] processJob: SUCCESS\n")
		return nil
	default:
		fmt.Printf("[  WARN ] processJob: Result channel full, dropping result for job %d\n", job.ID)
		return fmt.Errorf("result channel full")
	}
}

func (s *Scheduler) SetEventChannel(ch <-chan coretypes.ResultEvent) {
	s.eventCh = ch
}

func (s *Scheduler) GetResultChannel() <-chan types.OracleData {
	return s.resultCh
}

// GetHealthCheckFunc 헬스 체크 함수 반환
func (s *Scheduler) GetHealthCheckFunc() func(ctx context.Context) error {
	return func(ctx context.Context) error {
		// 최근 헬스 체크가 수행되었는지 확인
		if time.Since(s.lastHealthCheck) > 2*time.Minute {
			return fmt.Errorf("health check is stale (last check: %v)", s.lastHealthCheck)
		}

		// 결과 채널이 막혔는지 확인
		resultChLen := len(s.resultCh)
		resultChCap := cap(s.resultCh)
		if float64(resultChLen)/float64(resultChCap) > 0.9 {
			return fmt.Errorf("result channel is %d%% full", (resultChLen*100)/resultChCap)
		}

		// 실패한 이벤트가 너무 많은지 확인
		s.failedEventsMux.Lock()
		failedCount := len(s.failedEvents)
		s.failedEventsMux.Unlock()

		if failedCount > 500 {
			return fmt.Errorf("too many failed events: %d", failedCount)
		}

		return nil
	}
}

// GetActiveJobsCount activeJobs 개수 반환 (모니터링용)
func (s *Scheduler) GetActiveJobsCount() int {
	s.activeJobsMux.Lock()
	defer s.activeJobsMux.Unlock()
	return len(s.activeJobs)
}
