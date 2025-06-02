package scheduler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GPTx-global/guru/oracled/types"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

type Scheduler struct {
	eventCh       <-chan coretypes.ResultEvent
	activeJobs    map[string]*types.Job
	activeJobsMux sync.Mutex
	resultCh      chan types.OracleData
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		eventCh:       nil,
		activeJobs:    make(map[string]*types.Job),
		activeJobsMux: sync.Mutex{},
		resultCh:      make(chan types.OracleData, 64),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	go s.eventProcessor(ctx)
}

func (s *Scheduler) eventProcessor(ctx context.Context) {
	for {
		select {
		case event := <-s.eventCh:
			if !s.isOracleEvent(event) {
				continue
			}

			job, err := s.eventToJob(event)
			if err != nil || job == nil {
				fmt.Printf("Failed to parse oracle event: %v\n", err)
				continue
			}

			go s.processJob(ctx, job)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) processJob(ctx context.Context, job *types.Job) {
	s.activeJobsMux.Lock()
	if existingJob, exists := s.activeJobs[job.ID]; exists {
		job = existingJob
		job.Nonce++
	}
	s.activeJobs[job.ID] = job
	s.activeJobsMux.Unlock()

	executor := NewExecutor(ctx)
	oracleData, err := executor.ExecuteJob(job)
	if err != nil {
		fmt.Printf("Failed to execute job %s: %v\n", job.ID, err)
		return
	}

	s.resultCh <- *oracleData
}

func (s *Scheduler) eventToJob(event coretypes.ResultEvent) (*types.Job, error) {
	if event.Data == nil {
		return nil, nil
	}

	if strings.Contains(event.Query, "complete_oracle_data_set") {
		return s.parseOracleCompleteEvent(event)
	}

	eventData := event.Data
	switch eventData := eventData.(type) {
	case map[string]interface{}:
		if oracleRequest, ok := eventData["oracle_request"]; ok {
			return s.parseJobRequestEvent(oracleRequest)
		}

	default:
		// 일반적인 트랜잭션 이벤트에서 Oracle 관련 로그 파싱 (테스트용)
		return s.parseFromGenericEvent(event)
	}

	return nil, nil
}

func (s *Scheduler) parseJobRequestEvent(request interface{}) (*types.Job, error) {
	requestMap, ok := request.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid oracle request format")
	}

	id, ok := requestMap["id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing oracle request id")
	}

	dataSource, ok := requestMap["data_source"].(string)
	if !ok {
		return nil, fmt.Errorf("missing data source")
	}

	path, _ := requestMap["path"].(string)

	var delay time.Duration
	if delayStr, ok := requestMap["delay"].(string); ok {
		if delaySeconds, err := strconv.ParseInt(delayStr, 10, 64); err == nil {
			delay = time.Duration(delaySeconds) * time.Second
		}
	}

	return &types.Job{
		ID:    id,
		URL:   dataSource,
		Path:  path,
		Nonce: 0,
		Delay: delay,
	}, nil
}

func (s *Scheduler) parseOracleCompleteEvent(event coretypes.ResultEvent) (*types.Job, error) {
	// Oracle 완료 이벤트에서 Job ID와 완료된 nonce 추출
	eventStr := fmt.Sprintf("%v", event.Data)
	if strings.Contains(eventStr, "OracleId") {
		// 임시로 Job ID와 완료된 nonce 추출 (실제로는 이벤트 데이터에서 파싱)
		jobID := "BTC-PRICE"        // 실제로는 이벤트에서 추출
		completedNonce := uint64(2) // 실제로는 이벤트에서 추출된 완료된 nonce

		// 다음 nonce 계산 (완료된 nonce + 1)
		nextNonce := completedNonce + 1

		fmt.Printf("Oracle complete: Job %s nonce %d completed, next nonce: %d\n",
			jobID, completedNonce, nextNonce)

		// 다음 실행을 위한 Job 정보 반환 (nonce > 0이므로 지연 실행)
		return &types.Job{
			ID:    jobID,
			Nonce: nextNonce, // 완료된 nonce + 1 (지연 실행)
			// URL, Path, Delay는 activeJobs에서 복사
		}, nil
	}

	return nil, nil
}

// only for testing
func (s *Scheduler) parseFromGenericEvent(event coretypes.ResultEvent) (*types.Job, error) {
	if strings.Contains(event.Query, "NewBlock") {
		return &types.Job{
			ID:    fmt.Sprintf("auto-%d", 99),
			URL:   "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT",
			Path:  "price",
			Nonce: 0,
			Delay: 60 * time.Second,
		}, nil
	}

	return nil, nil
}

func (s *Scheduler) isOracleEvent(event coretypes.ResultEvent) bool {
	if event.Data == nil {
		return false
	}

	if strings.Contains(event.Query, "oracle") ||
		strings.Contains(event.Query, "complete_oracle_data_set") ||
		strings.Contains(event.Query, "NewBlock") {
		return true
	}

	return false
}

func (s *Scheduler) SetEventChannel(ch <-chan coretypes.ResultEvent) {
	s.eventCh = ch
}

func (s *Scheduler) GetResultChannel() <-chan types.OracleData {
	return s.resultCh
}
