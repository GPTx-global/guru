package client

import (
	"context"
	"testing"
	"time"

	"github.com/GPTx-global/guru/oracled/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

// ClientTestSuite는 Client 관련 통합 테스트를 위한 test suite입니다
type ClientTestSuite struct {
	suite.Suite
	client *Client
	ctx    context.Context
	cancel context.CancelFunc
}

func (suite *ClientTestSuite) SetupTest() {
	suite.client = NewClient()
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 30*time.Second)
}

func (suite *ClientTestSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
	}
}

func TestClientSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (suite *ClientTestSuite) TestClient_FullWorkflow() {
	// 전체 워크플로우 통합 테스트

	// 1. Job 채널에서 Job 수신 시뮬레이션
	job := &types.Job{
		ID:     1,
		URL:    "http://example.com/api",
		Path:   "$.data.price",
		Nonce:  1,
		Delay:  time.Second * 30,
		Status: "active",
	}

	// Job 전송
	go func() {
		select {
		case suite.client.jobCh <- job:
		case <-suite.ctx.Done():
		}
	}()

	// Job 수신 확인
	select {
	case receivedJob := <-suite.client.GetJobChannel():
		suite.Equal(job.ID, receivedJob.ID)
		suite.Equal(job.URL, receivedJob.URL)
	case <-time.After(time.Second):
		suite.Fail("Job was not received")
	}
}

func (suite *ClientTestSuite) TestClient_ConcurrentJobProcessing() {
	// 동시 Job 처리 테스트
	jobCount := 10
	jobs := make([]*types.Job, jobCount)

	for i := 0; i < jobCount; i++ {
		jobs[i] = &types.Job{
			ID:     uint64(i + 1),
			URL:    "http://example.com/api",
			Path:   "$.data.price",
			Nonce:  uint64(i + 1),
			Delay:  time.Second * 30,
			Status: "active",
		}
	}

	// 동시에 여러 Job 전송
	go func() {
		for _, job := range jobs {
			select {
			case suite.client.jobCh <- job:
			case <-suite.ctx.Done():
				return
			}
		}
	}()

	// 모든 Job 수신 확인
	receivedJobs := make([]*types.Job, 0, jobCount)
	for i := 0; i < jobCount; i++ {
		select {
		case job := <-suite.client.GetJobChannel():
			receivedJobs = append(receivedJobs, job)
		case <-time.After(5 * time.Second):
			suite.Fail("Not all jobs were received")
		}
	}

	suite.Len(receivedJobs, jobCount)
}

func (suite *ClientTestSuite) TestClient_ResultChannelHandling() {
	// 결과 채널 처리 테스트
	resultCh := make(chan types.OracleData, 10)
	suite.client.SetResultChannel(resultCh)

	// 테스트 데이터
	oracleData := types.OracleData{
		RequestID: 1,
		Data:      "100.50",
		Nonce:     1,
	}

	// 결과 전송
	go func() {
		select {
		case resultCh <- oracleData:
		case <-suite.ctx.Done():
		}
	}()

	// 결과 수신 확인
	select {
	case received := <-suite.client.resultCh:
		suite.Equal(oracleData.RequestID, received.RequestID)
		suite.Equal(oracleData.Data, received.Data)
		suite.Equal(oracleData.Nonce, received.Nonce)
	case <-time.After(time.Second):
		suite.Fail("Oracle data was not received")
	}
}

func (suite *ClientTestSuite) TestClient_ActiveJobsThreadSafety() {
	// activeJobs 동시성 안전성 테스트
	jobCount := 100

	// 동시에 여러 Job 추가
	for i := 0; i < jobCount; i++ {
		go func(id uint64) {
			job := &types.Job{
				ID:    id,
				URL:   "http://example.com/api",
				Nonce: id,
			}

			suite.client.activeJobsMux.Lock()
			suite.client.activeJobs[id] = job
			suite.client.activeJobsMux.Unlock()
		}(uint64(i))
	}

	// 잠시 대기 후 확인
	time.Sleep(100 * time.Millisecond)

	suite.client.activeJobsMux.Lock()
	suite.Len(suite.client.activeJobs, jobCount)
	suite.client.activeJobsMux.Unlock()
}

func (suite *ClientTestSuite) TestClient_EventToJobErrors() {
	// eventToJob 에러 시나리오 테스트
	tests := []struct {
		name      string
		event     coretypes.ResultEvent
		setupFunc func()
		wantErr   bool
	}{
		{
			name: "missing request_id",
			event: coretypes.ResultEvent{
				Data: tmtypes.EventDataTx{},
				Events: map[string][]string{
					"register_oracle_request_doc.status": {"active"},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid request_id format",
			event: coretypes.ResultEvent{
				Data: tmtypes.EventDataTx{},
				Events: map[string][]string{
					"register_oracle_request_doc.request_id": {"invalid"},
					"register_oracle_request_doc.status":     {"active"},
				},
			},
			wantErr: true,
		},
		{
			name: "job already exists",
			event: coretypes.ResultEvent{
				Data: tmtypes.EventDataTx{},
				Events: map[string][]string{
					"register_oracle_request_doc.request_id": {"1"},
					"register_oracle_request_doc.status":     {"active"},
				},
			},
			setupFunc: func() {
				suite.client.activeJobsMux.Lock()
				suite.client.activeJobs[1] = &types.Job{ID: 1}
				suite.client.activeJobsMux.Unlock()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			job, err := suite.client.eventToJob(tt.event)

			if tt.wantErr {
				suite.Error(err)
				suite.Nil(job)
			} else {
				suite.NoError(err)
				suite.NotNil(job)
			}
		})
	}
}

// 기존 개별 테스트들도 유지
func TestNewClient(t *testing.T) {
	client := NewClient()

	assert.NotNil(t, client)
	assert.NotNil(t, client.config)
	assert.NotNil(t, client.jobCh)
	assert.NotNil(t, client.activeJobs)
	assert.Equal(t, 256, cap(client.jobCh))
	assert.False(t, client.isConnected)
}

func TestClient_GetJobChannel(t *testing.T) {
	client := NewClient()

	ch := client.GetJobChannel()
	assert.NotNil(t, ch)

	// jobCh와 동일한 채널인지 확인
	assert.Equal(t, client.jobCh, ch)
}

func TestClient_SetResultChannel(t *testing.T) {
	client := NewClient()
	resultCh := make(chan types.OracleData, 10)

	client.SetResultChannel(resultCh)
	assert.Equal(t, resultCh, client.resultCh)
}

func TestClient_IsConnected(t *testing.T) {
	client := NewClient()

	// 초기 상태는 연결되지 않음
	assert.False(t, client.IsConnected())

	// isConnected만으로는 충분하지 않음 (rpcClient도 필요)
	client.isConnected = true
	assert.False(t, client.IsConnected()) // rpcClient가 nil이므로 false
}

func TestClient_EventToJob(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name    string
		event   coretypes.ResultEvent
		wantJob bool
		wantErr bool
	}{
		{
			name: "valid register oracle request",
			event: coretypes.ResultEvent{
				Data: tmtypes.EventDataTx{},
				Events: map[string][]string{
					"register_oracle_request_doc.request_id": {"1"},
					"register_oracle_request_doc.status":     {"active"},
				},
			},
			wantJob: false, // 실제로는 Tx 디코딩이 필요하므로 false
			wantErr: true,  // txDecoder에서 에러 발생
		},
		{
			name: "new block event with oracle completion",
			event: coretypes.ResultEvent{
				Data: tmtypes.EventDataNewBlock{},
				Events: map[string][]string{
					"complete_oracle_data_set.request_id": {"1"},
				},
			},
			wantJob: false, // activeJobs에 해당 ID가 없으므로 false
			wantErr: true,  // job not found error
		},
		{
			name: "unsupported event type",
			event: coretypes.ResultEvent{
				Data: "unsupported",
			},
			wantJob: false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job, err := client.eventToJob(tt.event)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantJob {
				assert.NotNil(t, job)
			} else {
				assert.Nil(t, job)
			}
		})
	}
}

func TestClient_JobChannelOperations(t *testing.T) {
	client := NewClient()

	// Job 채널 기본 상태 확인
	jobCh := client.GetJobChannel()
	assert.NotNil(t, jobCh)
	assert.Equal(t, 0, len(jobCh))
	assert.Equal(t, 256, cap(jobCh))

	// Job 전송 테스트 (실제로는 Client 내부에서만 전송)
	job := &types.Job{
		ID:     1,
		URL:    "http://example.com/api",
		Path:   "$.data.price",
		Nonce:  1,
		Delay:  time.Second * 30,
		Status: "active",
	}

	// client.jobCh에 직접 전송 (테스트 목적)
	select {
	case client.jobCh <- job:
		// 성공적으로 전송됨
	default:
		t.Fatal("Failed to send job to channel")
	}

	// 채널에서 수신 확인
	select {
	case receivedJob := <-jobCh:
		assert.Equal(t, job.ID, receivedJob.ID)
		assert.Equal(t, job.URL, receivedJob.URL)
		assert.Equal(t, job.Path, receivedJob.Path)
	default:
		t.Fatal("Failed to receive job from channel")
	}
}

func TestClient_ActiveJobsManagement(t *testing.T) {
	client := NewClient()

	job := &types.Job{
		ID:     1,
		URL:    "http://example.com/api",
		Path:   "$.data.price",
		Nonce:  1,
		Delay:  time.Second * 30,
		Status: "active",
	}

	// activeJobs 관리 테스트
	client.activeJobsMux.Lock()
	client.activeJobs[job.ID] = job
	client.activeJobsMux.Unlock()

	client.activeJobsMux.Lock()
	storedJob, exists := client.activeJobs[job.ID]
	client.activeJobsMux.Unlock()

	assert.True(t, exists)
	assert.Equal(t, job.ID, storedJob.ID)
	assert.Equal(t, job.URL, storedJob.URL)
}

func TestClient_HealthCheckFunc(t *testing.T) {
	client := NewClient()

	healthCheckFunc := client.GetHealthCheckFunc()
	assert.NotNil(t, healthCheckFunc)

	// 연결되지 않은 상태에서 헬스체크는 에러 반환
	err := healthCheckFunc(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestClient_ChannelBufferLimits(t *testing.T) {
	client := NewClient()

	// jobCh 버퍼 한계 테스트
	for i := 0; i < 256; i++ {
		job := &types.Job{ID: uint64(i + 1)}
		select {
		case client.jobCh <- job:
			// 성공
		default:
			t.Fatalf("Job channel should not be full at %d", i)
		}
	}

	// 257번째 job은 블로킹되어야 함
	extraJob := &types.Job{ID: 257}
	select {
	case client.jobCh <- extraJob:
		t.Fatal("Job channel should be full")
	default:
		// 예상된 동작
	}
}

func TestClient_EventProcessingEdgeCases(t *testing.T) {
	client := NewClient()

	// nil 이벤트 데이터
	event := coretypes.ResultEvent{Data: nil}
	job, err := client.eventToJob(event)
	assert.Error(t, err)
	assert.Nil(t, job)

	// 빈 이벤트 맵
	event = coretypes.ResultEvent{
		Data:   tmtypes.EventDataTx{},
		Events: map[string][]string{},
	}
	job, err = client.eventToJob(event)
	assert.Error(t, err)
	assert.Nil(t, job)
}
