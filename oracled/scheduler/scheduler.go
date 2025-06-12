package scheduler

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/GPTx-global/guru/oracled/types"
)

type Scheduler struct {
	jobCh         <-chan *types.Job
	activeJobs    map[uint64]*types.Job
	activeJobsMux sync.Mutex
	resultCh      chan types.OracleData

	// 기본적인 워커 풀
	workerPool *JobWorkerPool
}

func NewScheduler() *Scheduler {
	fmt.Printf("[ START ] NewScheduler\n")

	s := &Scheduler{
		jobCh:         nil,
		activeJobs:    make(map[uint64]*types.Job),
		activeJobsMux: sync.Mutex{},
		resultCh:      make(chan types.OracleData, 256),
		workerPool:    nil, // Start()에서 초기화
	}

	fmt.Printf("[  END  ] NewScheduler: SUCCESS - resultCh buffer size=256\n")
	return s
}

func (s *Scheduler) Start(ctx context.Context) {
	fmt.Printf("[ START ] Start\n")

	maxWorkers := runtime.NumCPU() * 2
	s.workerPool = NewJobWorkerPool(maxWorkers, s)
	s.workerPool.Start(ctx)

	go s.simpleJobProcessor(ctx)

	fmt.Printf("[  END  ] Start: SUCCESS - WorkerPool with %d workers\n", maxWorkers)
}

// 단순한 Job 처리기 - 복잡한 재시도 로직 제거
func (s *Scheduler) simpleJobProcessor(ctx context.Context) {
	fmt.Printf("[ START ] simpleJobProcessor\n")

	for {
		select {
		case job := <-s.jobCh:
			fmt.Printf("[ EVENT ] simpleJobProcessor: Received job - ID: %d, Nonce: %d\n", job.ID, job.Nonce)

			// Job을 활성 목록에 추가
			s.activeJobsMux.Lock()
			s.activeJobs[job.ID] = job
			s.activeJobsMux.Unlock()

			// WorkerPool에 Job 제출
			if !s.workerPool.SubmitJob(job) {
				fmt.Printf("[  WARN ] simpleJobProcessor: Worker pool queue full for job %d\n", job.ID)
			}

		case <-ctx.Done():
			fmt.Printf("[  END  ] simpleJobProcessor: SUCCESS - context cancelled\n")
			return
		}
	}
}

func (s *Scheduler) processJob(ctx context.Context, job *types.Job) error {
	fmt.Printf("[ START ] processJob - ID: %d, Nonce: %d, Status: %s\n", job.ID, job.Nonce, job.Status)

	executor := NewExecutor(ctx)

	// 간단한 실행
	oracleData, err := executor.ExecuteJob(job)
	if err != nil {
		fmt.Printf("[  END  ] processJob: ERROR - failed to execute job %d: %v\n", job.ID, err)

		// 기본적인 재시도 한 번만
		fmt.Printf("[  RETRY] processJob: Retrying job %d once\n", job.ID)
		time.Sleep(2 * time.Second)

		oracleData, err = executor.ExecuteJob(job)
		if err != nil {
			fmt.Printf("[  END  ] processJob: ERROR - job %d failed after retry: %v\n", job.ID, err)
			return fmt.Errorf("failed to execute job %d after retry: %w", job.ID, err)
		}
	}

	fmt.Printf("[  INFO ] processJob: Job executed successfully, sending to result channel\n")

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

func (s *Scheduler) SetJobChannel(ch <-chan *types.Job) {
	s.jobCh = ch
}

func (s *Scheduler) GetResultChannel() <-chan types.OracleData {
	return s.resultCh
}

// 간단한 헬스 체크 함수
func (s *Scheduler) GetHealthCheckFunc() func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if s.jobCh == nil {
			return fmt.Errorf("job channel not set")
		}

		if s.workerPool == nil {
			return fmt.Errorf("worker pool not initialized")
		}

		return nil
	}
}

func (s *Scheduler) GetActiveJobsCount() int {
	s.activeJobsMux.Lock()
	defer s.activeJobsMux.Unlock()
	return len(s.activeJobs)
}
