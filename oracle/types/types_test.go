package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/GPTx-global/guru/oracle/config"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TypesTestSuite struct {
	suite.Suite
}

func (suite *TypesTestSuite) SetupSuite() {
	// Load config to avoid nil pointer issues
	err := config.LoadConfig()
	suite.Require().NoError(err, "Failed to load config for testing")
}

func TestTypesTestSuite(t *testing.T) {
	suite.Run(t, new(TypesTestSuite))
}

func (suite *TypesTestSuite) TestJob_Structure() {
	// Given: Job 구조체
	job := &Job{
		ID:     123,
		URL:    "https://api.example.com/data",
		Path:   "price.usd",
		Nonce:  5,
		Delay:  30 * time.Second,
		Status: "ACTIVE",
	}

	// Then: 모든 필드가 정확히 설정됨
	suite.Equal(uint64(123), job.ID)
	suite.Equal("https://api.example.com/data", job.URL)
	suite.Equal("price.usd", job.Path)
	suite.Equal(uint64(5), job.Nonce)
	suite.Equal(30*time.Second, job.Delay)
	suite.Equal("ACTIVE", job.Status)
}

func (suite *TypesTestSuite) TestJobResult_Structure() {
	// Given: JobResult 구조체
	result := &JobResult{
		ID:    456,
		Data:  "1234.56",
		Nonce: 10,
	}

	// Then: 모든 필드가 정확히 설정됨
	suite.Equal(uint64(456), result.ID)
	suite.Equal("1234.56", result.Data)
	suite.Equal(uint64(10), result.Nonce)
}

func (suite *TypesTestSuite) TestMakeJob_FromOracleRequestDoc() {
	// Given: OracleRequestDoc 인스턴스
	doc := &oracletypes.OracleRequestDoc{
		RequestId: 100,
		Endpoints: []*oracletypes.OracleEndpoint{
			{
				Url:       "https://api.coinbase.com/v2/exchange-rates",
				ParseRule: "data.rates.USD",
			},
		},
		Nonce:  7,
		Period: 60,
		Status: oracletypes.RequestStatus_REQUEST_STATUS_ENABLED,
	}

	// When: MakeJob 호출
	jobs := MakeJobs(doc)

	// Then: Job이 올바르게 생성됨
	suite.NotNil(jobs)
	suite.Equal(1, len(jobs))
	job := jobs[0]
	suite.Equal(uint64(100), job.ID)
	suite.Equal("https://api.coinbase.com/v2/exchange-rates", job.URL)
	suite.Equal("data.rates.USD", job.Path)
	suite.Equal(uint64(7), job.Nonce)
	suite.Equal(60*time.Second, job.Delay)
	suite.Equal("REQUEST_STATUS_ENABLED", job.Status)
}

func (suite *TypesTestSuite) TestMakeJob_FromOracleRequestDoc_EmptyEndpoints() {
	// Given: 엔드포인트가 없는 OracleRequestDoc
	doc := &oracletypes.OracleRequestDoc{
		RequestId: 300,
		Endpoints: []*oracletypes.OracleEndpoint{},
		Nonce:     1,
		Period:    30,
		Status:    oracletypes.RequestStatus_REQUEST_STATUS_ENABLED,
	}

	// When/Then: MakeJob 호출 시 패닉 방지를 위한 테스트
	suite.Panics(func() {
		MakeJobs(doc)
	})
}

func (suite *TypesTestSuite) TestMakeJob_FromInvalidEvent() {
	// Given: 알 수 없는 타입의 이벤트
	invalidEvent := "invalid event type"

	// When: MakeJob 호출
	jobs := MakeJobs(invalidEvent)

	// Then: nil이 반환됨
	suite.Nil(jobs)
}

// 단위 테스트 함수들

func TestJobZeroValues(t *testing.T) {
	// Given: 제로값으로 초기화된 Job
	job := &Job{}

	// Then: 모든 필드가 제로값
	assert.Equal(t, uint64(0), job.ID)
	assert.Empty(t, job.URL)
	assert.Empty(t, job.Path)
	assert.Equal(t, uint64(0), job.Nonce)
	assert.Equal(t, time.Duration(0), job.Delay)
	assert.Empty(t, job.Status)
}

func TestJobResultZeroValues(t *testing.T) {
	// Given: 제로값으로 초기화된 JobResult
	result := &JobResult{}

	// Then: 모든 필드가 제로값
	assert.Equal(t, uint64(0), result.ID)
	assert.Empty(t, result.Data)
	assert.Equal(t, uint64(0), result.Nonce)
}

func TestMakeJobNilInput(t *testing.T) {
	// Given: nil 입력
	var nilInput interface{} = nil

	// When: MakeJob 호출
	job := MakeJobs(nilInput)

	// Then: nil이 반환됨
	assert.Nil(t, job)
}

func TestJobSerialization(t *testing.T) {
	// Given: Job 인스턴스
	original := &Job{
		ID:     999,
		URL:    "https://serialization.test",
		Path:   "data.value",
		Nonce:  15,
		Delay:  120 * time.Second,
		Status: "SERIALIZED",
	}

	// When: JSON 직렬화/역직렬화
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	var deserialized Job
	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	// Then: 원본과 동일
	assert.Equal(t, original.ID, deserialized.ID)
	assert.Equal(t, original.URL, deserialized.URL)
	assert.Equal(t, original.Path, deserialized.Path)
	assert.Equal(t, original.Nonce, deserialized.Nonce)
	assert.Equal(t, original.Delay, deserialized.Delay)
	assert.Equal(t, original.Status, deserialized.Status)
}

func TestJobResultSerialization(t *testing.T) {
	// Given: JobResult 인스턴스
	original := &JobResult{
		ID:    888,
		Data:  "serialized_data",
		Nonce: 25,
	}

	// When: JSON 직렬화/역직렬화
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	var deserialized JobResult
	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	// Then: 원본과 동일
	assert.Equal(t, original.ID, deserialized.ID)
	assert.Equal(t, original.Data, deserialized.Data)
	assert.Equal(t, original.Nonce, deserialized.Nonce)
}

func TestMakeJobWithValidOracleRequestDoc(t *testing.T) {
	// Load config first
	err := config.LoadConfig()
	require.NoError(t, err, "Failed to load config")

	// Given: 유효한 OracleRequestDoc
	doc := &oracletypes.OracleRequestDoc{
		RequestId: 12345,
		Endpoints: []*oracletypes.OracleEndpoint{
			{
				Url:       "https://api.test.com/price",
				ParseRule: "price.value",
			},
		},
		Nonce:  10,
		Period: 300,
		Status: oracletypes.RequestStatus_REQUEST_STATUS_ENABLED,
	}

	// When: MakeJob 호출
	jobs := MakeJobs(doc)

	// Then: 올바른 Job이 생성됨
	require.NotNil(t, jobs)
	require.Equal(t, 1, len(jobs))
	job := jobs[0]
	assert.Equal(t, uint64(12345), job.ID)
	assert.Equal(t, "https://api.test.com/price", job.URL)
	assert.Equal(t, "price.value", job.Path)
	assert.Equal(t, uint64(10), job.Nonce)
	assert.Equal(t, 300*time.Second, job.Delay)
	assert.Equal(t, "REQUEST_STATUS_ENABLED", job.Status)
}

func TestMakeJobWithMultipleEndpoints(t *testing.T) {
	// Load config first
	err := config.LoadConfig()
	require.NoError(t, err, "Failed to load config")

	// Given: 여러 엔드포인트를 가진 OracleRequestDoc
	doc := &oracletypes.OracleRequestDoc{
		RequestId: 54321,
		Endpoints: []*oracletypes.OracleEndpoint{
			{
				Url:       "https://api1.test.com",
				ParseRule: "price1",
			},
			{
				Url:       "https://api2.test.com",
				ParseRule: "price2",
			},
		},
		Nonce:  5,
		Period: 600,
		Status: oracletypes.RequestStatus_REQUEST_STATUS_PAUSED,
	}

	// When: MakeJob 호출
	jobs := MakeJobs(doc)

	// Then: 첫 번째 엔드포인트가 사용됨
	require.NotNil(t, jobs)
	require.Equal(t, 1, len(jobs))
	job := jobs[0]
	assert.Equal(t, "https://api1.test.com", job.URL)
	assert.Equal(t, "price1", job.Path)
	assert.Equal(t, "REQUEST_STATUS_PAUSED", job.Status)
}
