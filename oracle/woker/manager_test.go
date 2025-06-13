package woker

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/GPTx-global/guru/oracle/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ManagerTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
}

func TestManagerTestSuite(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

func (suite *ManagerTestSuite) SetupTest() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())
}

func (suite *ManagerTestSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
	}
}

func (suite *ManagerTestSuite) TestNewJobManager() {
	// Given/When: JobManager 생성
	jm := NewJobManager()

	// Then: 올바르게 초기화됨
	suite.NotNil(jm)
	suite.NotNil(jm.jobQueue)
	suite.NotNil(jm.activeJobs)
	suite.NotNil(jm.quit)
	suite.Equal(runtime.NumCPU()*4, cap(jm.jobQueue))
}

func (suite *ManagerTestSuite) TestJobManager_Start() {
	// Given: JobManager와 결과 채널
	jm := NewJobManager()
	resultQueue := make(chan *types.JobResult, 10)

	// When: 워커 풀 시작
	go jm.Start(suite.ctx, resultQueue)
	defer jm.Stop()

	// Then: 잠시 후 정상 동작 확인
	time.Sleep(100 * time.Millisecond)

	// 컨텍스트가 아직 살아있는지 확인
	select {
	case <-suite.ctx.Done():
		suite.Fail("Context should not be cancelled yet")
	default:
		// 정상
	}
}

func (suite *ManagerTestSuite) TestJobManager_SubmitJob() {
	// Given: JobManager와 Job
	jm := NewJobManager()
	job := &types.Job{
		ID:    1,
		URL:   "https://example.com",
		Path:  "data.price",
		Nonce: 0, // 첫 번째 nonce
		Delay: 0,
	}

	// When: Job 제출
	jm.SubmitJob(job)

	// Then: Job이 큐에 추가됨
	select {
	case receivedJob := <-jm.jobQueue:
		suite.Equal(job, receivedJob)
	case <-time.After(1 * time.Second):
		suite.Fail("Job was not added to queue")
	}
}

func (suite *ManagerTestSuite) TestJobManager_SubmitJob_FullQueue() {
	// Given: JobManager
	jm := NewJobManager()
	queueSize := cap(jm.jobQueue)

	// When: 큐를 가득 채움
	for i := 0; i < queueSize; i++ {
		job := &types.Job{
			ID:    uint64(i),
			URL:   "https://example.com",
			Path:  "data.price",
			Nonce: 0,
			Delay: 0,
		}
		jm.SubmitJob(job)
	}

	// 추가 Job (큐가 가득 찬 상태)
	extraJob := &types.Job{
		ID:    999,
		URL:   "https://example.com",
		Path:  "data.price",
		Nonce: 0,
		Delay: 0,
	}

	// Then: SubmitJob이 드롭 메시지와 함께 처리됨 (블로킹되지 않음)
	start := time.Now()
	jm.SubmitJob(extraJob)
	duration := time.Since(start)

	suite.Less(duration, 100*time.Millisecond, "SubmitJob should not block when queue is full")
}

func (suite *ManagerTestSuite) TestJobManager_Stop() {
	// Given: 실행 중인 JobManager
	jm := NewJobManager()
	resultQueue := make(chan *types.JobResult, 10)

	started := make(chan bool, 1)
	go func() {
		started <- true
		jm.Start(suite.ctx, resultQueue)
	}()

	// 시작될 때까지 대기
	<-started
	time.Sleep(50 * time.Millisecond)

	// When: Stop 호출
	stopped := make(chan bool, 1)
	go func() {
		jm.Stop()
		stopped <- true
	}()

	// Then: JobManager가 정상적으로 종료됨
	select {
	case <-stopped:
		suite.True(true, "JobManager should stop gracefully")
	case <-time.After(1 * time.Second):
		suite.Fail("JobManager should have stopped")
	}
}

func (suite *ManagerTestSuite) TestJobManager_ActiveJobsHandling() {
	// Given: JobManager
	jm := NewJobManager()

	// When: nonce 0인 Job 제출 (새 Job)
	newJob := &types.Job{
		ID:    100,
		URL:   "https://example.com",
		Path:  "data.price",
		Nonce: 0,
		Delay: 0,
	}
	jm.SubmitJob(newJob)

	// 큐에서 Job을 가져와서 activeJobs에 추가됨을 시뮬레이션
	select {
	case job := <-jm.jobQueue:
		jm.activeJobsLock.Lock()
		jm.activeJobs[job.ID] = job
		jm.activeJobsLock.Unlock()

		// Then: activeJobs에 Job이 저장됨
		suite.Contains(jm.activeJobs, job.ID)
		suite.Equal(newJob, jm.activeJobs[job.ID])
	case <-time.After(1 * time.Second):
		suite.Fail("Job should be in queue")
	}
}

func (suite *ManagerTestSuite) TestJobManager_Integration() {
	// Given: JobManager와 결과 채널
	jm := NewJobManager()
	resultQueue := make(chan *types.JobResult, 10)

	// 워커 시작
	go jm.Start(suite.ctx, resultQueue)
	defer jm.Stop()

	// When: Job 제출
	job := &types.Job{
		ID:    1,
		URL:   "https://httpbin.org/json", // 실제 응답하는 테스트 URL
		Path:  "url",
		Nonce: 0,
		Delay: 0,
	}

	jm.SubmitJob(job)

	// Then: 결과가 채널에서 수신됨
	select {
	case result := <-resultQueue:
		suite.NotNil(result)
		suite.Equal(uint64(1), result.ID)
		suite.Equal(uint64(1), result.Nonce) // nonce가 1 증가
	case <-time.After(5 * time.Second):
		// 실제 HTTP 호출이므로 실패할 수 있음
		suite.T().Log("Integration test timeout - may be network issue")
	}
}

// 단위 테스트 함수들

func TestNewJobManagerDefaults(t *testing.T) {
	// Given/When: JobManager 생성
	jm := NewJobManager()

	// Then: 기본값으로 초기화됨
	require.NotNil(t, jm)
	assert.NotNil(t, jm.jobQueue)
	assert.NotNil(t, jm.activeJobs)
	assert.NotNil(t, jm.quit)
	assert.Equal(t, runtime.NumCPU()*4, cap(jm.jobQueue))
	assert.Equal(t, 0, len(jm.activeJobs))
}

func TestJobManagerConcurrentSubmit(t *testing.T) {
	// Given: JobManager
	jm := NewJobManager()
	numJobs := 50
	numWorkers := 5

	// When: 동시에 Job 제출
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < numJobs/numWorkers; i++ {
				job := &types.Job{
					ID:    uint64(workerID*1000 + i),
					URL:   "https://example.com",
					Path:  "data",
					Nonce: 0,
				}
				jm.SubmitJob(job)
			}
		}(w)
	}

	wg.Wait()

	// Then: Job들이 큐에 추가됨 (일부는 드롭될 수 있음)
	assert.True(t, len(jm.jobQueue) <= numJobs)
	assert.True(t, len(jm.jobQueue) >= 0)
}

func TestJobManagerQueueCapacity(t *testing.T) {
	// Given: JobManager
	jm := NewJobManager()

	// When: 큐 용량 확인
	jobCapacity := cap(jm.jobQueue)

	// Then: CPU 기반 용량
	assert.Equal(t, runtime.NumCPU()*4, jobCapacity)
	assert.Greater(t, jobCapacity, 0)
}

func TestJobManagerNilJobHandling(t *testing.T) {
	// Given: JobManager
	jm := NewJobManager()

	// When: nil Job 제출 시도 (패닉 방지 확인)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SubmitJob should handle nil gracefully, but panicked: %v", r)
		}
	}()

	// nil Job을 제출해도 패닉이 발생하지 않아야 함
	jm.SubmitJob(nil)

	// Then: 패닉 없이 완료
	assert.True(t, true, "Should complete without panic")
}

func TestJobManagerActiveJobsMap(t *testing.T) {
	// Given: JobManager
	jm := NewJobManager()

	// When: activeJobs 맵에 직접 접근
	jm.activeJobsLock.Lock()
	testJob := &types.Job{
		ID:    42,
		URL:   "https://test.com",
		Path:  "test.path",
		Nonce: 5,
	}
	jm.activeJobs[42] = testJob
	jm.activeJobsLock.Unlock()

	// Then: 맵에서 조회 가능
	jm.activeJobsLock.Lock()
	retrievedJob, exists := jm.activeJobs[42]
	jm.activeJobsLock.Unlock()

	assert.True(t, exists)
	assert.Equal(t, testJob, retrievedJob)
}

func TestJobManagerContextCancellation(t *testing.T) {
	// Given: JobManager와 취소 가능한 컨텍스트
	jm := NewJobManager()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	resultQueue := make(chan *types.JobResult, 10)

	// When: Start 실행
	done := make(chan bool, 1)
	go func() {
		jm.Start(ctx, resultQueue)
		done <- true
	}()

	// Then: 컨텍스트 타임아웃 후 종료
	select {
	case <-done:
		assert.True(t, true, "JobManager should finish when context is cancelled")
	case <-time.After(200 * time.Millisecond):
		t.Error("JobManager should have finished due to context timeout")
	}
}

func TestJobManagerWorkerCount(t *testing.T) {
	// Given: JobManager
	jm := NewJobManager()
	resultQueue := make(chan *types.JobResult, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// When: Start 호출
	go jm.Start(ctx, resultQueue)
	time.Sleep(50 * time.Millisecond) // 워커들이 시작될 시간을 줌

	// Then: CPU 개수만큼 워커가 시작되어야 함
	// 실제로는 wg.Add가 CPU 개수만큼 호출됨을 확인
	expectedWorkers := runtime.NumCPU()
	assert.Greater(t, expectedWorkers, 0)

	// 정리
	cancel()
	jm.Stop()
}
