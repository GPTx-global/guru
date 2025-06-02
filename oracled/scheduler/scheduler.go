package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GPTx-global/guru/oracled/types"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

type Scheduler struct {
	// Client로부터 받는 채널 (이벤트 소비자)
	eventCh       <-chan coretypes.ResultEvent
	activeJobs    map[string]*types.Job
	activeJobsMux sync.RWMutex

	// Scheduler가 생성하고 소유하는 채널 (Oracle 데이터 생성자)
	oracleDataCh chan types.OracleData
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		// 이벤트는 Client에서 받으므로 nil로 초기화
		eventCh:       nil,
		activeJobs:    make(map[string]*types.Job),
		activeJobsMux: sync.RWMutex{},
		// Scheduler가 Oracle 데이터를 생성하므로 OracleDataCh 생성
		oracleDataCh: make(chan types.OracleData, 64),
	}
}

// Client의 EventCh를 설정 (읽기 전용으로 받음)
func (s *Scheduler) SetEventChannel(ch <-chan coretypes.ResultEvent) {
	s.eventCh = ch
}

// OracleDataCh의 읽기 전용 채널을 반환 (Client가 사용)
func (s *Scheduler) GetOracleDataChannel() <-chan types.OracleData {
	return s.oracleDataCh
}

func (s *Scheduler) Start() {
	go s.eventProcessor()
	fmt.Println("Scheduler started with event processor")
}

func (s *Scheduler) eventProcessor() {
	fmt.Println("Event processor started, waiting for events...")

	for event := range s.eventCh {
		// Oracle 관련 이벤트인지 확인
		if !s.isOracleEvent(event) {
			continue
		}

		fmt.Printf("Oracle event detected, parsing...\n")

		// 이벤트를 파싱하여 Job 추출
		job, err := s.eventToJob(event)
		if err != nil {
			fmt.Printf("Failed to parse oracle event: %v\n", err)
			continue
		}

		if job == nil {
			continue
		}

		// Job을 고루틴으로 즉시 처리
		go s.scheduleJob(job)
	}
}

func (s *Scheduler) scheduleJob(job *types.Job) {
	// 기존 작업이 있는 경우 정보 복사 (완료 이벤트의 경우)
	s.activeJobsMux.RLock()
	if existingJob, exists := s.activeJobs[job.ID]; exists {
		if job.URL == "" {
			job.URL = existingJob.URL
		}
		if job.Path == "" {
			job.Path = existingJob.Path
		}
		if job.Delay == 0 {
			job.Delay = existingJob.Delay
		}
	}
	s.activeJobsMux.RUnlock()

	// nonce에 따라 즉시 실행 vs 지연 실행 결정
	if job.Nonce == 0 {
		// nonce 0: 즉시 실행
		fmt.Printf("Processing immediate job: %s (nonce: %d)\n", job.ID, job.Nonce)
		s.executeJob(job)
	} else {
		// nonce > 0: delay 후 실행
		fmt.Printf("Processing delayed job: %s (nonce: %d)\n", job.ID, job.Nonce)
		if job.Delay > 0 {
			fmt.Printf("Waiting %v before executing job %s...\n", job.Delay, job.ID)
			time.Sleep(job.Delay)
		}
		s.executeJob(job)
	}
}

func (s *Scheduler) executeJob(job *types.Job) {
	// 활성 Job으로 등록/업데이트 (쓰기 락 사용)
	s.activeJobsMux.Lock()
	s.activeJobs[job.ID] = job
	s.activeJobsMux.Unlock()

	// 각 작업마다 새로운 Executor 생성
	executor := NewExecutor()

	// Executor에게 Job 실행 위임
	oracleData, err := executor.ExecuteJob(job)
	if err != nil {
		fmt.Printf("Failed to execute job %s: %v\n", job.ID, err)
		return
	}

	// Oracle 데이터를 채널로 전송
	select {
	case s.oracleDataCh <- *oracleData:
		fmt.Printf("Oracle data sent for job %s (nonce: %d): %s\n", job.ID, job.Nonce, oracleData.Data)
	case <-time.After(5 * time.Second):
		fmt.Printf("Timeout sending oracle data for job %s (nonce: %d)\n", job.ID, job.Nonce)
		return
	}

	fmt.Printf("Job %s completed successfully (nonce: %d)\n", job.ID, job.Nonce)
}

// 이벤트 파싱 메서드들 (EventParser에서 이동)
func (s *Scheduler) eventToJob(event coretypes.ResultEvent) (*types.Job, error) {
	if event.Data == nil {
		return nil, nil
	}

	// Oracle 완료 이벤트 체크
	if strings.Contains(event.Query, "complete_oracle_data_set") {
		return s.parseOracleCompleteEvent(event)
	}

	// 작업 요청 이벤트 체크
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

	// delay 파싱 (초 단위)
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
		Nonce: 0, // 새로운 작업은 nonce 0 (즉시 실행)
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

func (s *Scheduler) parseFromGenericEvent(event coretypes.ResultEvent) (*types.Job, error) {
	// 트랜잭션 로그에서 Oracle 요청을 찾는 로직
	// 현재는 모든 블록에 대해 임시 Job 생성 (테스트용)

	// 예시: 블록 높이를 기반으로 임시 Job 생성
	if strings.Contains(event.Query, "NewBlock") {
		height := s.extractBlockHeight(event)
		if height > 0 && height%10 == 0 { // 10블록마다 Oracle 작업 생성
			return &types.Job{
				ID:    fmt.Sprintf("auto-%d", height),
				URL:   "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT",
				Path:  "price",
				Nonce: 0,                // 새로운 작업은 즉시 실행
				Delay: 60 * time.Second, // 1분 대기
			}, nil
		}
	}

	return nil, nil
}

func (s *Scheduler) extractBlockHeight(event coretypes.ResultEvent) int64 {
	// 이벤트에서 블록 높이 추출
	if event.Data != nil {
		// 실제 구현에서는 이벤트 데이터를 파싱하여 높이를 얻어야 함
		// 현재는 임시로 1 반환
		return 1
	}
	return 0
}

func (s *Scheduler) isOracleEvent(event coretypes.ResultEvent) bool {
	// Oracle 관련 이벤트인지 확인
	if event.Data == nil {
		return false
	}

	// Oracle 관련 이벤트 감지 로직
	if strings.Contains(event.Query, "oracle") ||
		strings.Contains(event.Query, "complete_oracle_data_set") ||
		strings.Contains(event.Query, "NewBlock") { // 임시로 모든 블록 이벤트 포함
		return true
	}

	return false
}

// Job 관리 관련 메서드들 (동시 접근 보호)
func (s *Scheduler) GetActiveJobCount() int {
	s.activeJobsMux.RLock()
	defer s.activeJobsMux.RUnlock()
	return len(s.activeJobs)
}

func (s *Scheduler) GetActiveJobs() []string {
	s.activeJobsMux.RLock()
	defer s.activeJobsMux.RUnlock()

	jobs := make([]string, 0, len(s.activeJobs))
	for id := range s.activeJobs {
		jobs = append(jobs, id)
	}
	return jobs
}

func (s *Scheduler) IsJobActive(jobID string) bool {
	s.activeJobsMux.RLock()
	defer s.activeJobsMux.RUnlock()

	_, exists := s.activeJobs[jobID]
	return exists
}

func (s *Scheduler) GetJobNonce(jobID string) uint64 {
	s.activeJobsMux.RLock()
	defer s.activeJobsMux.RUnlock()

	if job, exists := s.activeJobs[jobID]; exists {
		return job.Nonce
	}
	return 0
}
