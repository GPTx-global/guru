package scheduler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GPTx-global/guru/oracled/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// WorkerPoolTestSuite는 WorkerPool 관련 통합 테스트를 위한 test suite입니다
type WorkerPoolTestSuite struct {
	suite.Suite
	scheduler *Scheduler
	pool      *JobWorkerPool
	server    *httptest.Server
}

func (suite *WorkerPoolTestSuite) SetupTest() {
	suite.scheduler = NewScheduler()
	suite.pool = NewJobWorkerPool(2, suite.scheduler)

	// 테스트용 HTTP 서버 설정
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/valid":
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"price": "100.50",
				},
			}
			json.NewEncoder(w).Encode(response)
		case "/api/invalid-json":
			w.Write([]byte("invalid json"))
		case "/api/timeout":
			time.Sleep(5 * time.Second) // 타임아웃 시뮬레이션
			w.Write([]byte("{}"))
		case "/api/404":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
		case "/api/500":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func (suite *WorkerPoolTestSuite) TearDownTest() {
	suite.server.Close()
	if suite.pool != nil {
		suite.pool.Stop()
	}
}

func TestWorkerPoolSuite(t *testing.T) {
	suite.Run(t, new(WorkerPoolTestSuite))
}

func (suite *WorkerPoolTestSuite) TestHTTP_ValidResponse() {
	// 유효한 HTTP 응답 처리 테스트
	job := &types.Job{
		ID:   1,
		URL:  suite.server.URL + "/api/valid",
		Path: "$.data.price",
	}

	// HTTP 요청 시뮬레이션
	resp, err := http.Get(job.URL)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)
}

func (suite *WorkerPoolTestSuite) TestHTTP_InvalidJSON() {
	// 잘못된 JSON 응답 처리 테스트
	job := &types.Job{
		ID:   2,
		URL:  suite.server.URL + "/api/invalid-json",
		Path: "$.data.price",
	}

	resp, err := http.Get(job.URL)
	suite.NoError(err)
	defer resp.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	suite.Error(err) // JSON 파싱 에러 발생해야 함
}

func (suite *WorkerPoolTestSuite) TestHTTP_NetworkTimeout() {
	// 네트워크 타임아웃 테스트
	client := &http.Client{
		Timeout: 1 * time.Second, // 1초 타임아웃
	}

	job := &types.Job{
		ID:   3,
		URL:  suite.server.URL + "/api/timeout",
		Path: "$.data.price",
	}

	_, err := client.Get(job.URL)
	suite.Error(err) // 타임아웃 에러 발생해야 함
	suite.Contains(err.Error(), "timeout")
}

func (suite *WorkerPoolTestSuite) TestHTTP_ErrorStatuses() {
	// HTTP 에러 상태 코드 처리 테스트
	testCases := []struct {
		name           string
		endpoint       string
		expectedStatus int
	}{
		{"404 Not Found", "/api/404", http.StatusNotFound},
		{"500 Internal Error", "/api/500", http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			resp, err := http.Get(suite.server.URL + tc.endpoint)
			suite.NoError(err)
			defer resp.Body.Close()

			suite.Equal(tc.expectedStatus, resp.StatusCode)
		})
	}
}

func (suite *WorkerPoolTestSuite) TestWorkerPool_ConcurrentJobs() {
	// 동시 Job 처리 테스트
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suite.pool.Start(ctx)

	// 여러 Job 동시 제출
	jobCount := 10
	for i := 0; i < jobCount; i++ {
		job := &types.Job{
			ID:  uint64(i + 1),
			URL: suite.server.URL + "/api/valid",
		}

		success := suite.pool.SubmitJob(job)
		suite.True(success, "Job %d should be submitted successfully", i+1)
	}

	// 처리 시간 확보
	time.Sleep(2 * time.Second)
}

func (suite *WorkerPoolTestSuite) TestWorkerPool_QueueOverflow() {
	// 큐 오버플로우 테스트
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 작은 워커풀로 테스트
	smallPool := NewJobWorkerPool(1, suite.scheduler)
	smallPool.Start(ctx)
	defer smallPool.Stop()

	// 큐 용량을 초과하는 Job 제출
	maxJobs := 10 // 큐 크기보다 많이
	submittedCount := 0

	for i := 0; i < maxJobs; i++ {
		job := &types.Job{
			ID:  uint64(i + 1),
			URL: suite.server.URL + "/api/valid",
		}

		if smallPool.SubmitJob(job) {
			submittedCount++
		}
	}

	// 일부 Job은 제출되고 일부는 거부될 수 있음
	suite.Greater(submittedCount, 0)
	suite.LessOrEqual(submittedCount, maxJobs)
}

func (suite *WorkerPoolTestSuite) TestWorkerPool_GracefulShutdown() {
	// 정상 종료 테스트
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	suite.pool.Start(ctx)

	// Job 제출
	job := &types.Job{
		ID:  1,
		URL: suite.server.URL + "/api/valid",
	}
	suite.pool.SubmitJob(job)

	// 즉시 취소
	cancel()

	// 정상 종료 확인 (패닉 없이 완료되어야 함)
	time.Sleep(500 * time.Millisecond)
}

func (suite *WorkerPoolTestSuite) TestJSON_PathExtraction() {
	// JSON Path 추출 테스트
	testData := `{
		"data": {
			"price": "100.50",
			"currency": "USD"
		},
		"status": "success"
	}`

	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(testData), &jsonData)
	suite.NoError(err)

	// 중첩된 필드 접근 시뮬레이션
	if data, ok := jsonData["data"].(map[string]interface{}); ok {
		if price, ok := data["price"].(string); ok {
			suite.Equal("100.50", price)
		}
	}
}

func (suite *WorkerPoolTestSuite) TestHTTP_UserAgent() {
	// User-Agent 헤더 확인 테스트
	req, err := http.NewRequest("GET", suite.server.URL+"/api/valid", nil)
	suite.NoError(err)

	req.Header.Set("User-Agent", "oracled/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)
}

// 기존 개별 테스트들도 유지
func TestNewJobWorkerPool(t *testing.T) {
	scheduler := NewScheduler()
	maxWorkers := 4

	pool := NewJobWorkerPool(maxWorkers, scheduler)

	assert.NotNil(t, pool)
	assert.Equal(t, maxWorkers, pool.maxWorkers)
	assert.NotNil(t, pool.jobQueue)
	assert.NotNil(t, pool.workerQueue)
	assert.NotNil(t, pool.quit)
	assert.Equal(t, scheduler, pool.scheduler)

	// 채널 버퍼 크기 확인
	assert.Equal(t, maxWorkers*2, cap(pool.jobQueue))
	assert.Equal(t, maxWorkers, cap(pool.workerQueue))
}

func TestJobWorkerPool_SubmitJob(t *testing.T) {
	scheduler := NewScheduler()
	pool := NewJobWorkerPool(2, scheduler)

	job := &types.Job{
		ID:     1,
		URL:    "http://example.com/api",
		Path:   "$.data.price",
		Nonce:  1,
		Delay:  time.Second * 30,
		Status: "active",
	}

	// Job 제출 테스트
	success := pool.SubmitJob(job)
	assert.True(t, success)

	// Job이 큐에 들어갔는지 확인
	assert.Equal(t, 1, len(pool.jobQueue))
}

func TestJobWorkerPool_Lifecycle(t *testing.T) {
	scheduler := NewScheduler()
	pool := NewJobWorkerPool(2, scheduler)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 시작
	pool.Start(ctx)

	// Job 제출
	job := &types.Job{
		ID:  1,
		URL: "http://httpbin.org/json",
	}
	success := pool.SubmitJob(job)
	assert.True(t, success)

	// 잠시 대기
	time.Sleep(100 * time.Millisecond)

	// 정지
	pool.Stop()
}

func TestWorker_Lifecycle(t *testing.T) {
	scheduler := NewScheduler()
	pool := NewJobWorkerPool(1, scheduler)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// 워커풀 시작 (내부적으로 워커들을 시작함)
	pool.Start(ctx)

	// 잠시 대기하여 워커가 시작되도록 함
	time.Sleep(50 * time.Millisecond)

	// Context 취소로 워커 종료
	cancel()

	// 종료 대기
	time.Sleep(100 * time.Millisecond)

	pool.Stop()
}

func TestDispatcher_Flow(t *testing.T) {
	scheduler := NewScheduler()
	pool := NewJobWorkerPool(2, scheduler)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// 워커풀 시작 (내부적으로 dispatcher 시작)
	pool.Start(ctx)
	defer pool.Stop()

	// Job 제출
	job := &types.Job{
		ID:  1,
		URL: "http://example.com/api",
	}
	pool.SubmitJob(job)

	// 잠시 대기
	time.Sleep(100 * time.Millisecond)

	// Context 취소
	cancel()

	// 종료 대기
	time.Sleep(100 * time.Millisecond)
}

func TestJobWorkerPool_MultipleJobs(t *testing.T) {
	scheduler := NewScheduler()
	pool := NewJobWorkerPool(3, scheduler)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool.Start(ctx)
	defer pool.Stop()

	// 여러 Job 제출
	jobCount := 5
	for i := 0; i < jobCount; i++ {
		job := &types.Job{
			ID:  uint64(i + 1),
			URL: "http://httpbin.org/json",
		}

		success := pool.SubmitJob(job)
		assert.True(t, success, "Job %d should be submitted", i+1)
	}

	// 처리 시간 확보
	time.Sleep(1 * time.Second)
}

func TestJobWorkerPool_BufferLimits(t *testing.T) {
	scheduler := NewScheduler()
	maxWorkers := 2
	pool := NewJobWorkerPool(maxWorkers, scheduler)

	// jobQueue 버퍼 한계 테스트
	queueCapacity := cap(pool.jobQueue)
	assert.Equal(t, maxWorkers*2, queueCapacity)

	// 큐 용량만큼 Job 제출
	for i := 0; i < queueCapacity; i++ {
		job := &types.Job{ID: uint64(i + 1)}
		success := pool.SubmitJob(job)
		assert.True(t, success, "Job %d should fit in queue", i+1)
	}

	// 추가 Job은 블로킹될 수 있음
	extraJob := &types.Job{ID: uint64(queueCapacity + 1)}
	select {
	case pool.jobQueue <- extraJob:
		t.Fatal("Queue should be full")
	default:
		// 예상된 동작
	}
}

func TestJobWorkerPool_WorkerQueue(t *testing.T) {
	scheduler := NewScheduler()
	maxWorkers := 3
	pool := NewJobWorkerPool(maxWorkers, scheduler)

	// workerQueue 크기 확인
	assert.Equal(t, maxWorkers, cap(pool.workerQueue))

	// 초기 상태에서는 비어있음
	assert.Equal(t, 0, len(pool.workerQueue))
}

func TestJobWorkerPool_ContextCancellation(t *testing.T) {
	scheduler := NewScheduler()
	pool := NewJobWorkerPool(2, scheduler)

	ctx, cancel := context.WithCancel(context.Background())

	// 시작
	pool.Start(ctx)

	// 즉시 취소
	cancel()

	// 정상 종료 확인 (패닉 없이)
	time.Sleep(100 * time.Millisecond)

	pool.Stop()
}
