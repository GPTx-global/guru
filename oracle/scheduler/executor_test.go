package scheduler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GPTx-global/guru/oracle/log"
	"github.com/GPTx-global/guru/oracle/types"
	"github.com/stretchr/testify/suite"
)

// ExecutorTestSuite defines the test suite for Executor functionality
type ExecutorTestSuite struct {
	suite.Suite
	testServer *httptest.Server
}

// SetupSuite runs once before all tests in the suite
func (suite *ExecutorTestSuite) SetupSuite() {
	// Initialize logging system
	log.InitLogger()

	// Set up test HTTP server
	suite.testServer = httptest.NewServer(http.HandlerFunc(suite.testHandler))
}

// TearDownSuite runs once after all tests in the suite complete
func (suite *ExecutorTestSuite) TearDownSuite() {
	if suite.testServer != nil {
		suite.testServer.Close()
	}
}

// testHandler is a test HTTP handler
func (suite *ExecutorTestSuite) testHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/simple":
		// Simple JSON object response
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"price":  50000.5,
				"volume": 1000000,
			},
			"status": "success",
		}
		json.NewEncoder(w).Encode(response)

	case "/nested":
		// Nested JSON object response
		response := map[string]interface{}{
			"result": map[string]interface{}{
				"market": map[string]interface{}{
					"btc": map[string]interface{}{
						"usd": 45000.75,
						"krw": 60000000,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)

	case "/array":
		// JSON response with array
		response := map[string]interface{}{
			"prices": []interface{}{
				map[string]interface{}{
					"symbol": "BTC",
					"price":  50000,
				},
				map[string]interface{}{
					"symbol": "ETH",
					"price":  3000,
				},
			},
		}
		json.NewEncoder(w).Encode(response)

	case "/array-root":
		// JSON response with array at root
		response := []interface{}{
			map[string]interface{}{
				"id":    1,
				"value": "first",
			},
			map[string]interface{}{
				"id":    2,
				"value": "second",
			},
		}
		json.NewEncoder(w).Encode(response)

	case "/invalid-json":
		// Invalid JSON response
		w.Write([]byte("invalid json"))

	case "/server-error":
		// Server error response
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))

	case "/not-found":
		// 404 error response
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))

	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}
}

// TestExecutorClient tests the executorClient function
func (suite *ExecutorTestSuite) TestExecutorClient() {
	client1 := executorClient()
	client2 := executorClient()

	// Verify singleton pattern
	suite.Same(client1, client2)
	suite.NotNil(client1)
	suite.Equal(30*time.Second, client1.Timeout)
}

// TestParseJSON_Object tests parseJSON function for object parsing
func (suite *ExecutorTestSuite) TestParseJSON_Object() {
	jsonData := []byte(`{"name": "test", "value": 123}`)

	result, err := parseJSON(jsonData)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("test", result["name"])
	suite.Equal(float64(123), result["value"])
}

// TestParseJSON_Array tests parseJSON function for array parsing
func (suite *ExecutorTestSuite) TestParseJSON_Array() {
	jsonData := []byte(`[{"id": 1, "name": "first"}, {"id": 2, "name": "second"}]`)

	result, err := parseJSON(jsonData)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(float64(1), result["id"])
	suite.Equal("first", result["name"])
}

// TestParseJSON_InvalidJSON tests parseJSON function for invalid JSON handling
func (suite *ExecutorTestSuite) TestParseJSON_InvalidJSON() {
	jsonData := []byte(`invalid json`)

	result, err := parseJSON(jsonData)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "failed to unmarshal JSON")
}

// TestParseJSON_EmptyArray tests parseJSON function for empty array handling
func (suite *ExecutorTestSuite) TestParseJSON_EmptyArray() {
	jsonData := []byte(`[]`)

	result, err := parseJSON(jsonData)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "response is not a JSON object or array")
}

// TestExtractDataByPath_Simple tests extractDataByPath function for simple paths
func (suite *ExecutorTestSuite) TestExtractDataByPath_Simple() {
	data := map[string]any{
		"price":  50000.5,
		"volume": 1000000,
	}

	result, err := extractDataByPath(data, "price")

	suite.NoError(err)
	suite.Equal("50000.5", result)
}

// TestExtractDataByPath_Nested tests extractDataByPath function for nested paths
func (suite *ExecutorTestSuite) TestExtractDataByPath_Nested() {
	data := map[string]any{
		"data": map[string]any{
			"market": map[string]any{
				"price": 45000.75,
			},
		},
	}

	result, err := extractDataByPath(data, "data.market.price")

	suite.NoError(err)
	suite.Equal("45000.75", result)
}

// TestExtractDataByPath_Array tests extractDataByPath function for array access
func (suite *ExecutorTestSuite) TestExtractDataByPath_Array() {
	data := map[string]any{
		"prices": []any{
			map[string]any{
				"symbol": "BTC",
				"value":  50000,
			},
			map[string]any{
				"symbol": "ETH",
				"value":  3000,
			},
		},
	}

	result, err := extractDataByPath(data, "prices.0.value")

	suite.NoError(err)
	suite.Equal("50000", result)
}

// TestExtractDataByPath_EmptyPath tests extractDataByPath function for empty paths
func (suite *ExecutorTestSuite) TestExtractDataByPath_EmptyPath() {
	data := map[string]any{"key": "value"}

	result, err := extractDataByPath(data, "")

	suite.Error(err)
	suite.Empty(result)
	suite.Contains(err.Error(), "path cannot be empty")
}

// TestExtractDataByPath_KeyNotFound tests extractDataByPath function for missing keys
func (suite *ExecutorTestSuite) TestExtractDataByPath_KeyNotFound() {
	data := map[string]any{"existing": "value"}

	result, err := extractDataByPath(data, "nonexistent")

	suite.Error(err)
	suite.Empty(result)
	suite.Contains(err.Error(), "key 'nonexistent' not found")
}

// TestExtractDataByPath_ArrayIndexOutOfBounds tests array index overflow
func (suite *ExecutorTestSuite) TestExtractDataByPath_ArrayIndexOutOfBounds() {
	data := map[string]any{
		"items": []any{"first", "second"},
	}

	result, err := extractDataByPath(data, "items.5")

	suite.Error(err)
	suite.Empty(result)
	suite.Contains(err.Error(), "array index 5 out of bounds")
}

// TestParseArrayIndex tests the parseArrayIndex function
func (suite *ExecutorTestSuite) TestParseArrayIndex() {
	// Normal case
	index, err := parseArrayIndex("123")
	suite.NoError(err)
	suite.Equal(123, index)

	// Zero case
	index, err = parseArrayIndex("0")
	suite.NoError(err)
	suite.Equal(0, index)

	// Empty string
	index, err = parseArrayIndex("")
	suite.Error(err)
	suite.Equal(-1, index)

	// Invalid character
	index, err = parseArrayIndex("12a")
	suite.Error(err)
	suite.Equal(-1, index)
}

// TestFetchRawData_Success tests the successful case of fetchRawData function
func (suite *ExecutorTestSuite) TestFetchRawData_Success() {
	url := suite.testServer.URL + "/simple"

	data, err := fetchRawData(url)

	suite.NoError(err)
	suite.NotNil(data)

	// Verify JSON parsing
	var result map[string]any
	err = json.Unmarshal(data, &result)
	suite.NoError(err)
	suite.Equal("success", result["status"])
}

// TestFetchRawData_NotFound tests the 404 error of fetchRawData function
func (suite *ExecutorTestSuite) TestFetchRawData_NotFound() {
	url := suite.testServer.URL + "/not-found"

	data, err := fetchRawData(url)

	suite.Error(err)
	suite.Nil(data)
	suite.Contains(err.Error(), "unexpected HTTP status")
}

// TestFetchRawData_InvalidURL tests the invalid URL of fetchRawData function
func (suite *ExecutorTestSuite) TestFetchRawData_InvalidURL() {
	// Use invalid protocol to trigger request creation failure
	url := "://invalid-url"

	data, err := fetchRawData(url)

	suite.Error(err)
	suite.Nil(data)
	suite.Contains(err.Error(), "failed to create HTTP request")
}

// TestExecuteJob_Success tests the successful case of executeJob function
func (suite *ExecutorTestSuite) TestExecuteJob_Success() {
	job := types.Job{
		ID:    1,
		URL:   suite.testServer.URL + "/simple",
		Path:  "data.price",
		Nonce: 1,
		Delay: 0,
	}

	result, err := executeJob(job)

	suite.NoError(err)
	suite.Equal(uint64(1), result.ID)
	suite.Equal(uint64(1), result.Nonce)
	suite.Equal("50000.5", result.Data)
}

// TestExecuteJob_NestedPath tests the nested path of executeJob function
func (suite *ExecutorTestSuite) TestExecuteJob_NestedPath() {
	job := types.Job{
		ID:    2,
		URL:   suite.testServer.URL + "/nested",
		Path:  "result.market.btc.usd",
		Nonce: 1,
		Delay: 0,
	}

	result, err := executeJob(job)

	suite.NoError(err)
	suite.Equal(uint64(2), result.ID)
	suite.Equal("45000.75", result.Data)
}

// TestExecuteJob_ArrayPath tests the array path of executeJob function
func (suite *ExecutorTestSuite) TestExecuteJob_ArrayPath() {
	job := types.Job{
		ID:    3,
		URL:   suite.testServer.URL + "/array",
		Path:  "prices.0.price",
		Nonce: 1,
		Delay: 0,
	}

	result, err := executeJob(job)

	suite.NoError(err)
	suite.Equal(uint64(3), result.ID)
	suite.Equal("50000", result.Data)
}

// TestExecuteJob_ArrayRoot tests the root array of executeJob function
func (suite *ExecutorTestSuite) TestExecuteJob_ArrayRoot() {
	job := types.Job{
		ID:    4,
		URL:   suite.testServer.URL + "/array-root",
		Path:  "value",
		Nonce: 1,
		Delay: 0,
	}

	result, err := executeJob(job)

	suite.NoError(err)
	suite.Equal(uint64(4), result.ID)
	suite.Equal("first", result.Data)
}

// TestExecuteJob_FetchError tests the data fetch error of executeJob function
func (suite *ExecutorTestSuite) TestExecuteJob_FetchError() {
	job := types.Job{
		ID:    5,
		URL:   suite.testServer.URL + "/not-found",
		Path:  "data.price",
		Nonce: 1,
		Delay: 0,
	}

	result, err := executeJob(job)

	suite.Error(err)
	suite.Equal(types.JobResult{}, result)
	suite.Contains(err.Error(), "failed to fetch raw data")
}

// TestExecuteJob_ParseError tests the JSON parsing error of executeJob function
func (suite *ExecutorTestSuite) TestExecuteJob_ParseError() {
	job := types.Job{
		ID:    6,
		URL:   suite.testServer.URL + "/invalid-json",
		Path:  "data.price",
		Nonce: 1,
		Delay: 0,
	}

	result, err := executeJob(job)

	suite.Error(err)
	suite.Equal(types.JobResult{}, result)
	suite.Contains(err.Error(), "failed to parse JSON")
}

// TestExecuteJob_PathError tests the path extraction error of executeJob function
func (suite *ExecutorTestSuite) TestExecuteJob_PathError() {
	job := types.Job{
		ID:    7,
		URL:   suite.testServer.URL + "/simple",
		Path:  "nonexistent.key",
		Nonce: 1,
		Delay: 0,
	}

	result, err := executeJob(job)

	suite.Error(err)
	suite.Equal(types.JobResult{}, result)
	suite.Contains(err.Error(), "failed to extract data by path")
}

// TestExecuteJob_WithDelay tests the delay functionality of executeJob function
func (suite *ExecutorTestSuite) TestExecuteJob_WithDelay() {
	job := types.Job{
		ID:    8,
		URL:   suite.testServer.URL + "/simple",
		Path:  "data.price",
		Nonce: 2, // Nonce > 1 should trigger delay
		Delay: 10 * time.Millisecond,
	}

	start := time.Now()
	result, err := executeJob(job)
	elapsed := time.Since(start)

	suite.NoError(err)
	suite.Equal(uint64(8), result.ID)
	suite.Equal("50000.5", result.Data)
	// Verify delay occurred (at least 5ms)
	suite.GreaterOrEqual(elapsed, 5*time.Millisecond)
}

// TestExecutorSuite runs the test suite
func TestExecutorSuite(t *testing.T) {
	suite.Run(t, new(ExecutorTestSuite))
}
