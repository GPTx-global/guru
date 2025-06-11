package scheduler

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/GPTx-global/guru/oracled/retry"
	"github.com/GPTx-global/guru/oracled/types"
)

type Scheduler struct {
	jobCh         <-chan *types.Job
	activeJobs    map[uint64]*types.Job
	activeJobsMux sync.Mutex
	resultCh      chan types.OracleData

	// 에러 처리를 위한 추가 필드
	failedJobs      []*types.Job
	failedJobsMux   sync.Mutex
	lastHealthCheck time.Time

	// 성능 최적화를 위한 워커 풀
	workerPool *JobWorkerPool
}

func NewScheduler() *Scheduler {
	fmt.Printf("[ START ] NewScheduler\n")

	s := &Scheduler{
		jobCh:           nil,
		activeJobs:      make(map[uint64]*types.Job),
		activeJobsMux:   sync.Mutex{},
		resultCh:        make(chan types.OracleData, 256), // 버퍼 크기 증가
		failedJobs:      make([]*types.Job, 0),
		lastHealthCheck: time.Now(),
		workerPool:      nil, // Start()에서 초기화
	}

	fmt.Printf("[  END  ] NewScheduler: SUCCESS - resultCh buffer size=256\n")
	return s
}

func (s *Scheduler) Start(ctx context.Context) {
	fmt.Printf("[ START ] Start\n")

	maxWorkers := runtime.NumCPU() * 2
	s.workerPool = NewJobWorkerPool(maxWorkers, s)
	s.workerPool.Start(ctx)

	go s.jobProcessor(ctx)
	go s.failedJobRetryProcessor(ctx)
	go s.healthMonitor(ctx)

	fmt.Printf("[  END  ] Start: SUCCESS - WorkerPool with %d workers\n", maxWorkers)
}

func (s *Scheduler) jobProcessor(ctx context.Context) {
	fmt.Printf("[ START ] jobProcessor\n")

	for {
		select {
		case job := <-s.jobCh:
			fmt.Printf("[ JOB   ] jobProcessor: Received job - ID: %d, Nonce: %d\n", job.ID, job.Nonce)

			// Job 처리를 재시도 로직으로 래핑
			err := retry.Do(ctx, retry.DefaultRetryConfig(),
				func() error {
					fmt.Printf("[  INFO ] jobProcessor: Processing job - ID: %d, Nonce: %d\n", job.ID, job.Nonce)

					// activeJobs에 job 추가/업데이트
					s.activeJobsMux.Lock()
					s.activeJobs[job.ID] = job
					s.activeJobsMux.Unlock()

					// 워커 풀에 job 제출 (고루틴 무제한 생성 방지)
					fmt.Printf("[  INFO ] jobProcessor: Submitting job to worker pool\n")
					if !s.workerPool.SubmitJob(job) {
						fmt.Printf("[  WARN ] jobProcessor: Worker pool queue full\n")
						return fmt.Errorf("worker pool queue full")
					}
					fmt.Printf("[  INFO ] jobProcessor: Successfully submitted job to worker pool\n")
					return nil
				},
				func(err error) bool {
					// 일시적인 시스템 에러만 재시도
					return retry.DefaultIsRetryable(err)
				},
			)

			if err != nil {
				fmt.Printf("[  WARN ] jobProcessor: Failed to process job after retries: %v\n", err)
				s.addFailedJob(job) // 실패한 job을 별도로 저장
			} else {
				fmt.Printf("[SUCCESS] jobProcessor: Job processed successfully - ID: %d\n", job.ID)
			}

		case <-ctx.Done():
			fmt.Printf("[  END  ] jobProcessor: SUCCESS - context cancelled\n")
			return
		}
	}
}

func (s *Scheduler) failedJobRetryProcessor(ctx context.Context) {
	fmt.Printf("[ START ] failedJobRetryProcessor\n")

	ticker := time.NewTicker(2 * time.Minute) // 2분마다 실패한 job 재처리
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.retryFailedJobs(ctx)
		case <-ctx.Done():
			fmt.Printf("[  END  ] failedJobRetryProcessor: SUCCESS - context cancelled\n")
			return
		}
	}
}

func (s *Scheduler) retryFailedJobs(ctx context.Context) {
	s.failedJobsMux.Lock()
	jobs := make([]*types.Job, len(s.failedJobs))
	copy(jobs, s.failedJobs)
	s.failedJobs = s.failedJobs[:0] // 리스트 클리어
	s.failedJobsMux.Unlock()

	if len(jobs) == 0 {
		return
	}

	fmt.Printf("[ RETRY ] retryFailedJobs: Retrying %d failed jobs\n", len(jobs))

	for _, job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// activeJobs에 job 추가/업데이트
		s.activeJobsMux.Lock()
		s.activeJobs[job.ID] = job
		s.activeJobsMux.Unlock()

		if !s.workerPool.SubmitJob(job) {
			fmt.Printf("[  WARN ] retryFailedJobs: Worker pool queue full, re-adding job to failed list\n")
			s.addFailedJob(job)
		} else {
			fmt.Printf("[  INFO ] retryFailedJobs: Successfully resubmitted job %d\n", job.ID)
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

	s.failedJobsMux.Lock()
	failedJobsCount := len(s.failedJobs)
	s.failedJobsMux.Unlock()

	resultChLen := len(s.resultCh)
	resultChCap := cap(s.resultCh)

	fmt.Printf("[ HEALTH] Scheduler status - ActiveJobs: %d, FailedJobs: %d, ResultCh: %d/%d\n",
		activeJobsCount, failedJobsCount, resultChLen, resultChCap)

	// 결과 채널이 거의 가득 찬 경우 경고
	if float64(resultChLen)/float64(resultChCap) > 0.8 {
		fmt.Printf("[  WARN ] healthMonitor: Result channel is %d%% full\n",
			(resultChLen*100)/resultChCap)
	}

	// 실패한 job이 너무 많은 경우 경고
	if failedJobsCount > 100 {
		fmt.Printf("[  WARN ] healthMonitor: Too many failed jobs: %d\n", failedJobsCount)
	}

	s.lastHealthCheck = time.Now()
}

func (s *Scheduler) addFailedJob(job *types.Job) {
	s.failedJobsMux.Lock()
	defer s.failedJobsMux.Unlock()

	// 중복 방지를 위해 기존에 있는지 확인
	for _, existingJob := range s.failedJobs {
		if existingJob.ID == job.ID {
			fmt.Printf("[  INFO ] addFailedJob: Job %d already in failed list\n", job.ID)
			return
		}
	}

	s.failedJobs = append(s.failedJobs, job)
	fmt.Printf("[  WARN ] addFailedJob: Added job %d to failed list (total: %d)\n", job.ID, len(s.failedJobs))
}

// processJobWithRetry: Job 처리 재시도 로직
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

// processJob: 실제 Job 실행 함수
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

func (s *Scheduler) SetJobChannel(ch <-chan *types.Job) {
	s.jobCh = ch
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

		// 실패한 job이 너무 많은지 확인
		s.failedJobsMux.Lock()
		failedCount := len(s.failedJobs)
		s.failedJobsMux.Unlock()

		if failedCount > 50 {
			return fmt.Errorf("too many failed jobs: %d", failedCount)
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
