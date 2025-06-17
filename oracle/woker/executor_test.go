package woker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GPTx-global/guru/oracle/types"
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

// Test parseJSON with valid JSON object
func (suite *ExecutorTestSuite) TestParseJSON_ValidObject() {
	testCases := []struct {
		name     string
		rawData  []byte
		expected map[string]any
	}{
		{
			name:    "simple object",
			rawData: []byte(`{"name":"John","age":30}`),
			expected: map[string]any{
				"name": "John",
				"age":  float64(30),
			},
		},
		{
			name:    "nested object",
			rawData: []byte(`{"data":{"price":100.5,"currency":"USD"}}`),
			expected: map[string]any{
				"data": map[string]any{
					"price":    float64(100.5),
					"currency": "USD",
				},
			},
		},
		{
			name:     "empty object",
			rawData:  []byte(`{}`),
			expected: map[string]any{},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result, err := parseJSON(tc.rawData)
			suite.NoError(err)
			suite.Equal(tc.expected, result)
		})
	}
}

// Test parseJSON with JSON array
func (suite *ExecutorTestSuite) TestParseJSON_Array() {
	testCases := []struct {
		name     string
		rawData  []byte
		expected map[string]any
	}{
		{
			name:    "array with object",
			rawData: []byte(`[{"name":"John","age":30},{"name":"Jane","age":25}]`),
			expected: map[string]any{
				"name": "John",
				"age":  float64(30),
			},
		},
		{
			name:    "array with single object",
			rawData: []byte(`[{"price":100.5}]`),
			expected: map[string]any{
				"price": float64(100.5),
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result, err := parseJSON(tc.rawData)
			suite.NoError(err)
			suite.Equal(tc.expected, result)
		})
	}
}

// Test parseJSON with invalid JSON
func (suite *ExecutorTestSuite) TestParseJSON_Invalid() {
	testCases := []struct {
		name    string
		rawData []byte
	}{
		{
			name:    "invalid JSON syntax",
			rawData: []byte(`{"name":"John",`),
		},
		{
			name:    "empty data",
			rawData: []byte(``),
		},
		{
			name:    "non-JSON string",
			rawData: []byte(`not json`),
		},
		{
			name:    "array with non-object",
			rawData: []byte(`["string", 123]`),
		},
		{
			name:    "empty array",
			rawData: []byte(`[]`),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result, err := parseJSON(tc.rawData)
			suite.Error(err)
			suite.Nil(result)
		})
	}
}

// Test extractDataByPath with valid paths
func (suite *ExecutorTestSuite) TestExtractDataByPath_ValidPaths() {
	data := map[string]any{
		"name": "John",
		"age":  float64(30),
		"address": map[string]any{
			"street": "123 Main St",
			"city":   "New York",
		},
		"numbers": []any{float64(10), float64(20), float64(30)},
		"items": []any{
			map[string]any{"id": float64(1), "name": "item1"},
			map[string]any{"id": float64(2), "name": "item2"},
		},
	}

	testCases := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple field",
			path:     "name",
			expected: "John",
		},
		{
			name:     "numeric field",
			path:     "age",
			expected: "30",
		},
		{
			name:     "nested field",
			path:     "address.street",
			expected: "123 Main St",
		},
		{
			name:     "nested deep field",
			path:     "address.city",
			expected: "New York",
		},
		{
			name:     "array element",
			path:     "numbers.0",
			expected: "10",
		},
		{
			name:     "array last element",
			path:     "numbers.2",
			expected: "30",
		},
		{
			name:     "nested array field",
			path:     "items.0.name",
			expected: "item1",
		},
		{
			name:     "nested array numeric field",
			path:     "items.1.id",
			expected: "2",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result, err := extractDataByPath(data, tc.path)
			suite.NoError(err)
			suite.Equal(tc.expected, result)
		})
	}
}

// Test extractDataByPath with invalid paths
func (suite *ExecutorTestSuite) TestExtractDataByPath_InvalidPaths() {
	data := map[string]any{
		"name": "John",
		"age":  float64(30),
		"address": map[string]any{
			"street": "123 Main St",
		},
		"numbers": []any{float64(10), float64(20)},
	}

	testCases := []struct {
		name string
		path string
	}{
		{
			name: "empty path",
			path: "",
		},
		{
			name: "non-existent field",
			path: "nonexistent",
		},
		{
			name: "non-existent nested field",
			path: "address.nonexistent",
		},
		{
			name: "array index out of bounds",
			path: "numbers.5",
		},
		{
			name: "invalid array index",
			path: "numbers.abc",
		},
		{
			name: "path through non-object",
			path: "name.something",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result, err := extractDataByPath(data, tc.path)
			suite.Error(err)
			suite.Empty(result)
		})
	}
}

// Test parseArrayIndex function
func (suite *ExecutorTestSuite) TestParseArrayIndex() {
	testCases := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		{
			name:     "valid index 0",
			input:    "0",
			expected: 0,
			hasError: false,
		},
		{
			name:     "valid index 123",
			input:    "123",
			expected: 123,
			hasError: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: -1,
			hasError: true,
		},
		{
			name:     "invalid character",
			input:    "12a",
			expected: -1,
			hasError: true,
		},
		{
			name:     "negative sign",
			input:    "-1",
			expected: -1,
			hasError: true,
		},
		{
			name:     "special characters",
			input:    "1.5",
			expected: -1,
			hasError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result, err := parseArrayIndex(tc.input)
			if tc.hasError {
				suite.Error(err)
				suite.Equal(-1, result)
			} else {
				suite.NoError(err)
				suite.Equal(tc.expected, result)
			}
		})
	}
}

// Test executeJob function logic (without actual HTTP calls)
func (suite *ExecutorTestSuite) TestExecuteJob_DelayLogic() {
	// Test job with nonce > 1 (should have delay)
	suite.Run("job with nonce > 1", func() {
		job := &types.Job{
			ID:    1,
			URL:   "http://invalid-url.com", // Will fail HTTP, but tests delay logic
			Path:  "data.value",
			Type:  types.Register,
			Nonce: 2, // > 1, should trigger delay
			Delay: time.Millisecond,
		}

		start := time.Now()
		result := executeJob(job)
		elapsed := time.Since(start)

		// Should have waited for delay
		suite.GreaterOrEqual(elapsed, time.Millisecond)
		suite.Nil(result) // Should be nil due to HTTP failure, but delay was tested
	})

	// Test job with nonce <= 1 (no delay)
	suite.Run("job with nonce <= 1", func() {
		job := &types.Job{
			ID:    1,
			URL:   "http://invalid-url.com", // Will fail HTTP
			Path:  "data.value",
			Type:  types.Register,
			Nonce: 1,           // <= 1, no delay
			Delay: time.Second, // Long delay, but shouldn't be applied
		}

		start := time.Now()
		result := executeJob(job)
		elapsed := time.Since(start)

		// Should not have waited for delay
		suite.Less(elapsed, 500*time.Millisecond)
		suite.Nil(result) // Should be nil due to HTTP failure
	})
}

// Test edge cases for extractDataByPath
func (suite *ExecutorTestSuite) TestExtractDataByPath_EdgeCases() {
	// Test with complex nested structure
	suite.Run("complex nested structure", func() {
		data := map[string]any{
			"level1": map[string]any{
				"level2": []any{
					map[string]any{
						"level3": map[string]any{
							"value": "found",
						},
					},
				},
			},
		}

		result, err := extractDataByPath(data, "level1.level2.0.level3.value")
		suite.NoError(err)
		suite.Equal("found", result)
	})

	// Test with mixed types in path
	suite.Run("mixed types", func() {
		data := map[string]any{
			"string":  "text",
			"number":  float64(42),
			"boolean": true,
			"null":    nil,
		}

		// String value
		result, err := extractDataByPath(data, "string")
		suite.NoError(err)
		suite.Equal("text", result)

		// Number value
		result, err = extractDataByPath(data, "number")
		suite.NoError(err)
		suite.Equal("42", result)

		// Boolean value
		result, err = extractDataByPath(data, "boolean")
		suite.NoError(err)
		suite.Equal("true", result)

		// Null value
		result, err = extractDataByPath(data, "null")
		suite.NoError(err)
		suite.Equal("<nil>", result)
	})
}

// Test parseJSON edge cases
func (suite *ExecutorTestSuite) TestParseJSON_EdgeCases() {
	// Test with array containing mixed types
	suite.Run("array with mixed types", func() {
		rawData := []byte(`[{"valid":"object"}, "string", 123]`)
		result, err := parseJSON(rawData)
		suite.NoError(err)
		suite.Equal(map[string]any{"valid": "object"}, result)
	})

	// Test with deeply nested object
	suite.Run("deeply nested object", func() {
		rawData := []byte(`{"a":{"b":{"c":{"d":"deep"}}}}`)
		expected := map[string]any{
			"a": map[string]any{
				"b": map[string]any{
					"c": map[string]any{
						"d": "deep",
					},
				},
			},
		}
		result, err := parseJSON(rawData)
		suite.NoError(err)
		suite.Equal(expected, result)
	})

	// Test with special characters
	suite.Run("special characters", func() {
		rawData := []byte(`{"unicode":"测试","emoji":"😀","escape":"line1\nline2"}`)
		result, err := parseJSON(rawData)
		suite.NoError(err)
		suite.Equal("测试", result["unicode"])
		suite.Equal("😀", result["emoji"])
		suite.Equal("line1\nline2", result["escape"])
	})
}

// Test concurrent access to executorClient
func (suite *ExecutorTestSuite) TestExecutorClient_Singleton() {
	// Test that multiple calls to executorClient return the same instance
	clients := make([]*http.Client, 10)
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(index int) {
			clients[index] = executorClient()
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// All clients should be the same instance (singleton)
	firstClient := clients[0]
	for i := 1; i < 10; i++ {
		suite.Same(firstClient, clients[i])
	}

	// Verify client configuration
	suite.NotNil(firstClient)
	suite.Equal(30*time.Second, firstClient.Timeout)
}

// Test parseArrayIndex with edge cases
func (suite *ExecutorTestSuite) TestParseArrayIndex_EdgeCases() {
	testCases := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		{
			name:     "single digit",
			input:    "7",
			expected: 7,
			hasError: false,
		},
		{
			name:     "multiple digits",
			input:    "123456",
			expected: 123456,
			hasError: false,
		},
		{
			name:     "leading zeros",
			input:    "007",
			expected: 7,
			hasError: false,
		},
		{
			name:     "just zero",
			input:    "000",
			expected: 0,
			hasError: false,
		},
		{
			name:     "mixed invalid chars",
			input:    "1a2b3",
			expected: -1,
			hasError: true,
		},
		{
			name:     "space character",
			input:    "1 2",
			expected: -1,
			hasError: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result, err := parseArrayIndex(tc.input)
			if tc.hasError {
				suite.Error(err)
				suite.Equal(-1, result)
			} else {
				suite.NoError(err)
				suite.Equal(tc.expected, result)
			}
		})
	}
}

// Test type assertions in extractDataByPath
func (suite *ExecutorTestSuite) TestExtractDataByPath_TypeAssertions() {
	// Test invalid type casting scenarios
	data := map[string]any{
		"string":    "not_an_object",
		"number":    123,
		"boolean":   true,
		"array":     []any{1, 2, 3},
		"object":    map[string]any{"field": "value"},
		"null":      nil,
		"interface": interface{}(map[string]any{"nested": "value"}),
	}

	testCases := []struct {
		name     string
		path     string
		hasError bool
	}{
		{
			name:     "traverse through string",
			path:     "string.field",
			hasError: true,
		},
		{
			name:     "traverse through number",
			path:     "number.field",
			hasError: true,
		},
		{
			name:     "traverse through boolean",
			path:     "boolean.field",
			hasError: true,
		},
		{
			name:     "traverse through null",
			path:     "null.field",
			hasError: true,
		},
		{
			name:     "valid array access",
			path:     "array.1",
			hasError: false,
		},
		{
			name:     "valid object access",
			path:     "object.field",
			hasError: false,
		},
		{
			name:     "interface cast to map",
			path:     "interface.nested",
			hasError: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			_, err := extractDataByPath(data, tc.path)
			if tc.hasError {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
		})
	}
}
