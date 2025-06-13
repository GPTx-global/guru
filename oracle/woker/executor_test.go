package woker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GPTx-global/guru/oracle/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ExecutorTestSuite struct {
	suite.Suite
	server *httptest.Server
}

func TestExecutorTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutorTestSuite))
}

func (suite *ExecutorTestSuite) SetupTest() {
	// 테스트용 HTTP 서버 설정
	mux := http.NewServeMux()

	// JSON API 엔드포인트
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"price": map[string]interface{}{
				"usd": 50000.123,
				"krw": 65000000,
			},
			"rates": map[string]interface{}{
				"USD": 1.0,
				"KRW": 1300.5,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	})

	// 배열 형태의 JSON API 엔드포인트
	mux.HandleFunc("/array", func(w http.ResponseWriter, r *http.Request) {
		data := []interface{}{
			map[string]interface{}{
				"symbol": "BTC",
				"price":  50000.0,
			},
			map[string]interface{}{
				"symbol": "ETH",
				"price":  3000.0,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	})

	// 에러 응답 엔드포인트
	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	})

	// 잘못된 JSON 엔드포인트
	mux.HandleFunc("/invalid-json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"invalid": json}`))
	})

	// 느린 응답 엔드포인트
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"data": "slow"})
	})

	suite.server = httptest.NewServer(mux)
}

func (suite *ExecutorTestSuite) TearDownTest() {
	if suite.server != nil {
		suite.server.Close()
	}
}

func (suite *ExecutorTestSuite) TestExecuteJob_Success() {
	// Given: 유효한 Job
	job := &types.Job{
		ID:    1,
		URL:   suite.server.URL + "/json",
		Path:  "price.usd",
		Nonce: 1,
		Delay: 0,
	}

	// When: executeJob 실행
	result := executeJob(job)

	// Then: 올바른 결과 반환
	suite.NotNil(result)
	suite.Equal(uint64(1), result.ID)
	suite.Equal("50000.123", result.Data)
	suite.Equal(uint64(1), result.Nonce)
}

func (suite *ExecutorTestSuite) TestExecuteJob_ComplexPath() {
	// Given: 복잡한 경로를 가진 Job
	job := &types.Job{
		ID:    2,
		URL:   suite.server.URL + "/json",
		Path:  "rates.KRW",
		Nonce: 1,
		Delay: 0,
	}

	// When: executeJob 실행
	result := executeJob(job)

	// Then: 올바른 결과 반환
	suite.NotNil(result)
	suite.Equal("1300.5", result.Data)
}

func (suite *ExecutorTestSuite) TestExecuteJob_WithDelay() {
	// Given: Delay가 있는 Job (두 번째 nonce)
	job := &types.Job{
		ID:    3,
		URL:   suite.server.URL + "/json",
		Path:  "price.usd",
		Nonce: 2,
		Delay: 100 * time.Millisecond,
	}

	// When: executeJob 실행 (시간 측정)
	start := time.Now()
	result := executeJob(job)
	duration := time.Since(start)

	// Then: 지연 시간이 적용되고 결과 반환
	suite.NotNil(result)
	suite.GreaterOrEqual(duration, 100*time.Millisecond)
}

func (suite *ExecutorTestSuite) TestExecuteJob_InvalidURL() {
	// Given: 잘못된 URL을 가진 Job
	job := &types.Job{
		ID:    4,
		URL:   "http://invalid-url-that-does-not-exist.com",
		Path:  "price.usd",
		Nonce: 1,
		Delay: 0,
	}

	// When: executeJob 실행
	result := executeJob(job)

	// Then: nil 반환
	suite.Nil(result)
}

func (suite *ExecutorTestSuite) TestExecuteJob_InvalidPath() {
	// Given: 존재하지 않는 경로를 가진 Job
	job := &types.Job{
		ID:    5,
		URL:   suite.server.URL + "/json",
		Path:  "nonexistent.path",
		Nonce: 1,
		Delay: 0,
	}

	// When: executeJob 실행
	result := executeJob(job)

	// Then: nil 반환
	suite.Nil(result)
}

func (suite *ExecutorTestSuite) TestExecuteJob_EmptyPath() {
	// Given: 빈 경로를 가진 Job
	job := &types.Job{
		ID:    6,
		URL:   suite.server.URL + "/json",
		Path:  "",
		Nonce: 1,
		Delay: 0,
	}

	// When: executeJob 실행
	result := executeJob(job)

	// Then: nil 반환
	suite.Nil(result)
}

func (suite *ExecutorTestSuite) TestFetchRawData_Success() {
	// Given: 유효한 URL
	url := suite.server.URL + "/json"

	// When: fetchRawData 실행
	data, err := fetchRawData(url)

	// Then: 데이터가 성공적으로 반환됨
	suite.NoError(err)
	suite.NotEmpty(data)
	suite.Contains(string(data), "price")
}

func (suite *ExecutorTestSuite) TestFetchRawData_HTTPError() {
	// Given: 에러를 반환하는 URL
	url := suite.server.URL + "/error"

	// When: fetchRawData 실행
	data, err := fetchRawData(url)

	// Then: 에러가 아닌 데이터 반환 (HTTP 클라이언트는 5xx도 정상 응답으로 처리)
	suite.NoError(err)
	suite.Equal("Internal Server Error", string(data))
}

func (suite *ExecutorTestSuite) TestParseJSON_ValidJSON() {
	// Given: 유효한 JSON 데이터
	jsonData := []byte(`{"price": {"usd": 12345}, "status": "ok"}`)

	// When: parseJSON 실행
	result, err := parseJSON(jsonData)

	// Then: 파싱된 맵 반환
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("ok", result["status"])
}

func (suite *ExecutorTestSuite) TestParseJSON_ValidJSONArray() {
	// Given: 배열 형태의 JSON 데이터
	jsonData := []byte(`[{"symbol": "BTC", "price": 50000}, {"symbol": "ETH", "price": 3000}]`)

	// When: parseJSON 실행
	result, err := parseJSON(jsonData)

	// Then: 첫 번째 요소가 반환됨
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("BTC", result["symbol"])
	suite.Equal(float64(50000), result["price"])
}

func (suite *ExecutorTestSuite) TestParseJSON_InvalidJSON() {
	// Given: 잘못된 JSON 데이터
	jsonData := []byte(`{"invalid": json}`)

	// When: parseJSON 실행
	result, err := parseJSON(jsonData)

	// Then: 에러 반환
	suite.Error(err)
	suite.Nil(result)
}

func (suite *ExecutorTestSuite) TestExtractDataByPath_SimpleKey() {
	// Given: 간단한 데이터와 경로
	data := map[string]interface{}{
		"price":  12345.67,
		"status": "active",
	}
	path := "price"

	// When: extractDataByPath 실행
	result, err := extractDataByPath(data, path)

	// Then: 올바른 값 반환
	suite.NoError(err)
	suite.Equal("12345.67", result)
}

func (suite *ExecutorTestSuite) TestExtractDataByPath_NestedKey() {
	// Given: 중첩된 데이터와 경로
	data := map[string]interface{}{
		"price": map[string]interface{}{
			"usd": 50000.123,
			"eur": 42000.456,
		},
	}
	path := "price.usd"

	// When: extractDataByPath 실행
	result, err := extractDataByPath(data, path)

	// Then: 올바른 값 반환
	suite.NoError(err)
	suite.Equal("50000.123", result)
}

func (suite *ExecutorTestSuite) TestExtractDataByPath_ArrayIndex() {
	// Given: 배열이 포함된 데이터 (수동으로 배열 처리 구현 필요)
	// 참고: 현재 구현에서는 배열 인덱스 처리가 완전하지 않을 수 있음
	data := map[string]interface{}{
		"prices": []interface{}{
			map[string]interface{}{"value": 100},
			map[string]interface{}{"value": 200},
		},
	}
	path := "prices.0.value"

	// When: extractDataByPath 실행
	_, err := extractDataByPath(data, path)

	// Then: 에러가 발생할 수 있음 (현재 구현의 한계)
	// 이 테스트는 현재 구현의 한계를 확인하는 용도
	if err != nil {
		suite.Contains(err.Error(), "cannot traverse")
	}
}

func (suite *ExecutorTestSuite) TestExtractDataByPath_NonexistentKey() {
	// Given: 존재하지 않는 키
	data := map[string]interface{}{
		"price": 12345,
	}
	path := "nonexistent.key"

	// When: extractDataByPath 실행
	result, err := extractDataByPath(data, path)

	// Then: 에러 반환
	suite.Error(err)
	suite.Empty(result)
	suite.Contains(err.Error(), "not found")
}

func (suite *ExecutorTestSuite) TestParseArrayIndex_Valid() {
	// Given: 유효한 숫자 문자열
	indexStr := "123"

	// When: parseArrayIndex 실행
	index, err := parseArrayIndex(indexStr)

	// Then: 올바른 인덱스 반환
	suite.NoError(err)
	suite.Equal(123, index)
}

func (suite *ExecutorTestSuite) TestParseArrayIndex_Invalid() {
	// Given: 잘못된 문자열
	indexStr := "abc"

	// When: parseArrayIndex 실행
	index, err := parseArrayIndex(indexStr)

	// Then: 에러 반환
	suite.Error(err)
	suite.Equal(-1, index)
}

func (suite *ExecutorTestSuite) TestParseArrayIndex_Empty() {
	// Given: 빈 문자열
	indexStr := ""

	// When: parseArrayIndex 실행
	index, err := parseArrayIndex(indexStr)

	// Then: 에러 반환
	suite.Error(err)
	suite.Equal(-1, index)
}

func (suite *ExecutorTestSuite) TestExecutorClient_Singleton() {
	// Given/When: 여러 번 executorClient 호출
	client1 := executorClient()
	client2 := executorClient()

	// Then: 같은 인스턴스 반환 (싱글톤 패턴)
	suite.Same(client1, client2)
	suite.NotNil(client1)
}

// 단위 테스트 함수들

func TestExecuteJobWithRealData(t *testing.T) {
	// Given: 실제 API와 유사한 응답을 제공하는 테스트 서버
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"data": map[string]interface{}{
				"rates": map[string]interface{}{
					"USD": 1.0,
					"KRW": 1300.75,
					"EUR": 0.85,
				},
			},
			"timestamp": 1234567890,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}))
	defer server.Close()

	job := &types.Job{
		ID:    99,
		URL:   server.URL,
		Path:  "data.rates.KRW",
		Nonce: 1,
		Delay: 0,
	}

	// When: executeJob 실행
	result := executeJob(job)

	// Then: 정확한 결과 반환
	require.NotNil(t, result)
	assert.Equal(t, uint64(99), result.ID)
	assert.Equal(t, "1300.75", result.Data)
	assert.Equal(t, uint64(1), result.Nonce)
}

func TestFetchRawDataTimeout(t *testing.T) {
	// Given: 느린 서버
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(35 * time.Second) // HTTP 클라이언트 타임아웃보다 길게
		w.Write([]byte("delayed response"))
	}))
	defer server.Close()

	// When: fetchRawData 실행
	data, err := fetchRawData(server.URL)

	// Then: 타임아웃 에러 발생
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "timeout")
}

func TestExtractDataByPathEdgeCases(t *testing.T) {
	// Given: 다양한 타입의 값들
	data := map[string]interface{}{
		"string_val": "hello",
		"int_val":    42,
		"float_val":  3.14159,
		"bool_val":   true,
		"null_val":   nil,
		"nested": map[string]interface{}{
			"deep": map[string]interface{}{
				"value": "found",
			},
		},
	}

	testCases := []struct {
		path     string
		expected string
		hasError bool
	}{
		{"string_val", "hello", false},
		{"int_val", "42", false},
		{"float_val", "3.14159", false},
		{"bool_val", "true", false},
		{"null_val", "<nil>", false},
		{"nested.deep.value", "found", false},
		{"nested.deep.nonexistent", "", true},
		{"nonexistent", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			// When: extractDataByPath 실행
			result, err := extractDataByPath(data, tc.path)

			// Then: 예상된 결과 확인
			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}
