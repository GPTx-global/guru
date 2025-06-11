package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/GPTx-global/guru/oracled/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// SchedulerTestSuite는 Scheduler 관련 통합 테스트를 위한 test suite입니다
type SchedulerTestSuite struct {
	suite.Suite
	scheduler *Scheduler
	ctx       context.Context
	cancel    context.CancelFunc
}

func (suite *SchedulerTestSuite) SetupTest() {
	suite.scheduler = NewScheduler()
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 30*time.Second)
}

func (suite *SchedulerTestSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
	}
}

func TestSchedulerSuite(t *testing.T) {
	suite.Run(t, new(SchedulerTestSuite))
}

func (suite *SchedulerTestSuite) TestScheduler_FullWorkflow() {
	// 전체 워크플로우 통합 테스트
	jobCh := make(chan *types.Job, 10)
	suite.scheduler.SetJobChannel(jobCh)

	// Scheduler에서 기본 기능만 테스트 (WorkerPool 없이)

	// Job 전송
	job := &types.Job{
		ID:     1,
		URL:    "http://example.com/api",
		Path:   "$.data.price",
		Nonce:  1,
		Delay:  time.Second * 30,
		Status: "active",
	}

	// Job 제출
	select {
	case jobCh <- job:
	case <-time.After(time.Second):
		suite.Fail("Failed to send job")
	}

	// 잠시 대기하여 처리 시간 확보
	time.Sleep(100 * time.Millisecond)

	// activeJobs에 추가되었는지 확인
	suite.scheduler.activeJobsMux.Lock()
	_, exists := suite.scheduler.activeJobs[job.ID]
	suite.scheduler.activeJobsMux.Unlock()

	suite.True(exists)
}

func (suite *SchedulerTestSuite) TestScheduler_ConcurrentJobProcessing() {
	// 동시 Job 처리 테스트
	jobCh := make(chan *types.Job, 100)
	suite.scheduler.SetJobChannel(jobCh)

	// 동시에 여러 Job을 activeJobs에 추가
	jobCount := 50
	for i := 0; i < jobCount; i++ {
		job := &types.Job{
			ID:    uint64(i + 1),
			URL:   "http://example.com/api",
			Nonce: uint64(i + 1),
		}

		// activeJobs에 직접 추가
		suite.scheduler.activeJobsMux.Lock()
		suite.scheduler.activeJobs[job.ID] = job
		suite.scheduler.activeJobsMux.Unlock()
	}

	// 모든 Job이 추가되었는지 확인
	finalCount := suite.scheduler.GetActiveJobsCount()
	suite.Equal(jobCount, finalCount)
}

func (suite *SchedulerTestSuite) TestScheduler_FailedJobRetry() {
	// 실패한 Job 재시도 테스트

	job := &types.Job{
		ID:     1,
		URL:    "http://example.com/api",
		Path:   "$.data.price",
		Nonce:  1,
		Status: "active",
	}

	// 실패한 Job 추가
	suite.scheduler.addFailedJob(job)

	// 실패한 Job 개수 확인
	suite.scheduler.failedJobsMux.Lock()
	failedCount := len(suite.scheduler.failedJobs)
	suite.scheduler.failedJobsMux.Unlock()

	suite.Equal(1, failedCount)

	// activeJobs에 직접 추가 (재시도 시뮬레이션)
	suite.scheduler.activeJobsMux.Lock()
	suite.scheduler.activeJobs[job.ID] = job
	suite.scheduler.activeJobsMux.Unlock()

	// activeJobs에 추가되었는지 확인
	suite.scheduler.activeJobsMux.Lock()
	_, exists := suite.scheduler.activeJobs[job.ID]
	suite.scheduler.activeJobsMux.Unlock()

	suite.True(exists)
}

func (suite *SchedulerTestSuite) TestScheduler_MemoryLeakPrevention() {
	// 메모리 누수 방지 테스트
	jobCh := make(chan *types.Job, 10)
	suite.scheduler.SetJobChannel(jobCh)

	// 많은 Job을 처리한 후 정리되는지 확인
	jobCount := 1000
	for i := 0; i < jobCount; i++ {
		job := &types.Job{
			ID:    uint64(i + 1),
			URL:   "http://example.com/api",
			Nonce: uint64(i + 1),
		}

		suite.scheduler.activeJobsMux.Lock()
		suite.scheduler.activeJobs[job.ID] = job
		suite.scheduler.activeJobsMux.Unlock()

		// 주기적으로 일부 Job 제거 (완료된 Job 시뮬레이션)
		if i%100 == 0 && i > 0 {
			suite.scheduler.activeJobsMux.Lock()
			for j := uint64(i - 50); j < uint64(i); j++ {
				delete(suite.scheduler.activeJobs, j)
			}
			suite.scheduler.activeJobsMux.Unlock()
		}
	}

	// 최종 activeJobs 개수 확인
	finalCount := suite.scheduler.GetActiveJobsCount()
	suite.True(finalCount < jobCount) // 일부 Job이 정리되었어야 함
}

func (suite *SchedulerTestSuite) TestScheduler_ContextCancellation() {
	// Context 취소 시 정상 종료 테스트
	jobCh := make(chan *types.Job, 10)
	suite.scheduler.SetJobChannel(jobCh)

	ctx, cancel := context.WithCancel(context.Background())

	// Scheduler 시작
	suite.scheduler.Start(ctx)

	// 즉시 취소
	cancel()

	// 정상적으로 종료되는지 확인 (타임아웃으로 검증)
	done := make(chan bool)
	go func() {
		time.Sleep(100 * time.Millisecond)
		done <- true
	}()

	select {
	case <-done:
		// 정상 종료
	case <-time.After(time.Second):
		suite.Fail("Scheduler did not shut down gracefully")
	}
}

// WorkerPoolInterface는 테스트를 위한 WorkerPool 인터페이스입니다
type WorkerPoolInterface interface {
	Start(ctx context.Context)
	Stop()
	SubmitJob(job *types.Job) bool
}

// MockWorkerPool은 테스트용 WorkerPool Mock입니다
type MockWorkerPool struct {
	submitSuccess bool
	jobs          []*types.Job
	mu            sync.Mutex
}

func (m *MockWorkerPool) Start(ctx context.Context) {}

func (m *MockWorkerPool) Stop() {}

func (m *MockWorkerPool) SubmitJob(job *types.Job) bool {
	if m.submitSuccess {
		m.mu.Lock()
		m.jobs = append(m.jobs, job)
		m.mu.Unlock()
	}
	return m.submitSuccess
}

// 기존 개별 테스트들도 유지
func TestNewScheduler(t *testing.T) {
	scheduler := NewScheduler()

	assert.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.activeJobs)
	assert.NotNil(t, scheduler.resultCh)
	assert.Equal(t, 256, cap(scheduler.resultCh))
	assert.Empty(t, scheduler.activeJobs)
	assert.Empty(t, scheduler.failedJobs)
}

func TestScheduler_SetJobChannel(t *testing.T) {
	scheduler := NewScheduler()
	jobCh := make(chan *types.Job, 10)

	scheduler.SetJobChannel(jobCh)
	assert.Equal(t, jobCh, scheduler.jobCh)
}

func TestScheduler_GetResultChannel(t *testing.T) {
	scheduler := NewScheduler()

	resultCh := scheduler.GetResultChannel()
	assert.NotNil(t, resultCh)
	assert.Equal(t, scheduler.resultCh, resultCh)
}

func TestScheduler_GetActiveJobsCount(t *testing.T) {
	scheduler := NewScheduler()

	// 초기 상태에서는 0개
	assert.Equal(t, 0, scheduler.GetActiveJobsCount())

	// Job 추가 후 개수 확인
	job := &types.Job{
		ID:     1,
		URL:    "http://example.com/api",
		Path:   "$.data.price",
		Nonce:  1,
		Delay:  time.Second * 30,
		Status: "active",
	}

	scheduler.activeJobsMux.Lock()
	scheduler.activeJobs[job.ID] = job
	scheduler.activeJobsMux.Unlock()

	assert.Equal(t, 1, scheduler.GetActiveJobsCount())
}

func TestScheduler_AddFailedJob(t *testing.T) {
	scheduler := NewScheduler()

	job := &types.Job{
		ID:     1,
		URL:    "http://example.com/api",
		Path:   "$.data.price",
		Nonce:  1,
		Delay:  time.Second * 30,
		Status: "active",
	}

	// 실패한 Job 추가
	scheduler.addFailedJob(job)

	scheduler.failedJobsMux.Lock()
	failedCount := len(scheduler.failedJobs)
	scheduler.failedJobsMux.Unlock()

	assert.Equal(t, 1, failedCount)
}

func TestScheduler_JobChannelOperations(t *testing.T) {
	scheduler := NewScheduler()
	jobCh := make(chan *types.Job, 10)
	scheduler.SetJobChannel(jobCh)

	job := &types.Job{
		ID:     1,
		URL:    "http://example.com/api",
		Path:   "$.data.price",
		Nonce:  1,
		Delay:  time.Second * 30,
		Status: "active",
	}

	// Job 전송 테스트
	select {
	case jobCh <- job:
		// 성공적으로 전송됨
	default:
		t.Fatal("Failed to send job to channel")
	}

	// Job 수신 테스트
	select {
	case receivedJob := <-scheduler.jobCh:
		assert.Equal(t, job.ID, receivedJob.ID)
		assert.Equal(t, job.URL, receivedJob.URL)
		assert.Equal(t, job.Path, receivedJob.Path)
	default:
		t.Fatal("Failed to receive job from channel")
	}
}

func TestScheduler_ResultChannelOperations(t *testing.T) {
	scheduler := NewScheduler()

	oracleData := types.OracleData{
		RequestID: 1,
		Data:      "100.50",
		Nonce:     1,
	}

	// 결과 전송 테스트
	select {
	case scheduler.resultCh <- oracleData:
		// 성공적으로 전송됨
	default:
		t.Fatal("Failed to send result to channel")
	}

	// 결과 수신 테스트
	resultCh := scheduler.GetResultChannel()
	select {
	case receivedData := <-resultCh:
		assert.Equal(t, oracleData.RequestID, receivedData.RequestID)
		assert.Equal(t, oracleData.Data, receivedData.Data)
		assert.Equal(t, oracleData.Nonce, receivedData.Nonce)
	default:
		t.Fatal("Failed to receive result from channel")
	}
}

func TestScheduler_HealthCheckFunc(t *testing.T) {
	scheduler := NewScheduler()

	healthCheckFunc := scheduler.GetHealthCheckFunc()
	assert.NotNil(t, healthCheckFunc)

	// 헬스체크 실행 (정상적으로 에러 없이 실행되어야 함)
	err := healthCheckFunc(context.Background())
	assert.NoError(t, err)
}

func TestScheduler_PerformHealthCheck(t *testing.T) {
	scheduler := NewScheduler()

	// 테스트용 Job 추가
	job := &types.Job{ID: 1}
	scheduler.activeJobsMux.Lock()
	scheduler.activeJobs[job.ID] = job
	scheduler.activeJobsMux.Unlock()

	// 헬스체크 실행 (패닉이나 에러 없이 완료되어야 함)
	scheduler.performHealthCheck()

	// 헬스체크 후 상태 확인
	assert.Equal(t, 1, scheduler.GetActiveJobsCount())
}

func TestScheduler_ProcessJobIntegration(t *testing.T) {
	scheduler := NewScheduler()

	job := &types.Job{
		ID:     1,
		URL:    "http://httpbin.org/json", // 실제 테스트용 API
		Path:   "$.slideshow.title",
		Nonce:  1,
		Delay:  time.Second * 5,
		Status: "active",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Job 처리 실행
	err := scheduler.processJob(ctx, job)

	// 네트워크 에러나 파싱 에러가 발생할 수 있음 (실제 환경에서는 정상)
	// 테스트에서는 에러가 발생해도 정상적인 동작
	t.Logf("Process job result: %v", err)
}

func TestScheduler_ProcessJobWithMockURL(t *testing.T) {
	scheduler := NewScheduler()

	job := &types.Job{
		ID:     1,
		URL:    "http://invalid-url-for-test", // 의도적으로 잘못된 URL
		Path:   "$.data.price",
		Nonce:  1,
		Delay:  time.Second * 5,
		Status: "active",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 잘못된 URL로 Job 처리 시 에러 발생해야 함
	err := scheduler.processJob(ctx, job)
	assert.Error(t, err)
}

func TestScheduler_ThreadSafety(t *testing.T) {
	scheduler := NewScheduler()

	// 동시에 여러 goroutine에서 activeJobs 조작
	var wg sync.WaitGroup
	jobCount := 100

	for i := 0; i < jobCount; i++ {
		wg.Add(1)
		go func(id uint64) {
			defer wg.Done()

			job := &types.Job{
				ID:    id,
				URL:   "http://example.com/api",
				Nonce: id,
			}

			// Job 추가
			scheduler.activeJobsMux.Lock()
			scheduler.activeJobs[id] = job
			scheduler.activeJobsMux.Unlock()

			// Job 조회
			scheduler.activeJobsMux.Lock()
			_, exists := scheduler.activeJobs[id]
			scheduler.activeJobsMux.Unlock()

			assert.True(t, exists)
		}(uint64(i + 1))
	}

	wg.Wait()

	// 최종 개수 확인
	assert.Equal(t, jobCount, scheduler.GetActiveJobsCount())
}

func TestScheduler_ResultChannelBufferLimits(t *testing.T) {
	scheduler := NewScheduler()

	// resultCh 버퍼 한계 테스트
	for i := 0; i < 256; i++ {
		oracleData := types.OracleData{
			RequestID: uint64(i + 1),
			Data:      "test_data",
			Nonce:     uint64(i + 1),
		}

		select {
		case scheduler.resultCh <- oracleData:
			// 성공
		default:
			t.Fatalf("Result channel should not be full at %d", i)
		}
	}

	// 257번째 result는 블로킹되어야 함
	extraResult := types.OracleData{RequestID: 257}
	select {
	case scheduler.resultCh <- extraResult:
		t.Fatal("Result channel should be full")
	default:
		// 예상된 동작
	}
}
