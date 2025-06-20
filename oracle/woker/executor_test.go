package woker

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/stretchr/testify/suite"
)

type ExecutorSuite struct {
	suite.Suite
	server *httptest.Server
	mu     sync.Mutex
}

func (s *ExecutorSuite) SetupTest() {
	log.InitLogger()
	config.SetForTesting("test-chain", "", "test", os.TempDir(), keyring.BackendTest, "", 0)

	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		switch r.URL.Path {
		case "/success":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{"price":123.45},"tickers":[{"name":"BTC","last":60000}]}`))
		case "/invalid-json":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":"invalid`))
		case "/server-error":
			w.WriteHeader(http.StatusInternalServerError)
		case "/array":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"price": 456.78}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Reset httpClient for each test to ensure isolation
	once = sync.Once{}
	httpClient = nil
	_ = executorClient()
}

func (s *ExecutorSuite) TearDownTest() {
	s.server.Close()
}

func TestExecutorSuite(t *testing.T) {
	suite.Run(t, new(ExecutorSuite))
}

// func (s *ExecutorSuite) TestExecuteJob() {
// 	testCases := []struct {
// 		name          string
// 		job           *types.Job
// 		mock          func()
// 		expectError   bool
// 		expectedData  string
// 		expectedNonce uint64
// 	}{
// 		{
// 			name: "Success case",
// 			job: &types.Job{
// 				ID:    1,
// 				Nonce: 1,
// 				URL:   s.server.URL + "/success",
// 				Path:  "data.price",
// 			},
// 			expectError:   false,
// 			expectedData:  "123.45",
// 			expectedNonce: 1,
// 		},
// 		{
// 			name: "Fetch raw data failed",
// 			job: &types.Job{
// 				ID:    2,
// 				Nonce: 1,
// 				URL:   s.server.URL + "/server-error",
// 				Path:  "data.price",
// 			},
// 			expectError: true,
// 		},
// 		{
// 			name: "Parse JSON failed",
// 			job: &types.Job{
// 				ID:    3,
// 				Nonce: 1,
// 				URL:   s.server.URL + "/invalid-json",
// 				Path:  "data.price",
// 			},
// 			expectError: true,
// 		},
// 		{
// 			name: "Extract data by path failed",
// 			job: &types.Job{
// 				ID:    4,
// 				Nonce: 1,
// 				URL:   s.server.URL + "/success",
// 				Path:  "data.nonexistent",
// 			},
// 			expectError: true,
// 		},
// 		{
// 			name: "Delay for nonce > 1",
// 			job: &types.Job{
// 				ID:    5,
// 				Nonce: 2,
// 				URL:   s.server.URL + "/success",
// 				Path:  "data.price",
// 				Delay: 10 * time.Millisecond,
// 			},
// 			expectError:   false,
// 			expectedData:  "123.45",
// 			expectedNonce: 2,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		s.Run(tc.name, func() {
// 			if tc.mock != nil {
// 				tc.mock()
// 			}

// 			startTime := time.Now()
// 			result, err := executeJob(tc.job)
// 			elapsed := time.Since(startTime)

// 			if tc.job.Nonce > 1 {
// 				s.Require().GreaterOrEqual(elapsed, tc.job.Delay)
// 			}

// 			if tc.expectError {
// 				s.Require().Error(err)
// 				s.Require().Nil(result)
// 			} else {
// 				s.Require().NoError(err)
// 				s.Require().NotNil(result)
// 				s.Require().Equal(tc.job.ID, result.ID)
// 				s.Require().Equal(tc.expectedData, result.Data)
// 				s.Require().Equal(tc.expectedNonce, result.Nonce)
// 			}
// 		})
// 	}
// }

// func (s *ExecutorSuite) TestFetchRawData() {
// 	testCases := []struct {
// 		name          string
// 		url           string
// 		expectError   bool
// 		expectedBody  []byte
// 		clientFactory func() *http.Client
// 	}{
// 		{
// 			name:         "Success",
// 			url:          s.server.URL + "/success",
// 			expectError:  false,
// 			expectedBody: []byte(`{"data":{"price":123.45},"tickers":[{"name":"BTC","last":60000}]}`),
// 		},
// 		{
// 			name:        "Server error",
// 			url:         s.server.URL + "/server-error",
// 			expectError: true,
// 		},
// 		{
// 			name:        "Invalid URL",
// 			url:         "invalid-url",
// 			expectError: true,
// 		},
// 		{
// 			name:        "Request creation fails",
// 			url:         string([]byte{0x7f}), // Invalid character in URL
// 			expectError: true,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		s.Run(tc.name, func() {
// 			if tc.clientFactory != nil {
// 				httpClient = tc.clientFactory()
// 			}

// 			body, err := fetchRawData(tc.url)

// 			if tc.expectError {
// 				s.Require().Error(err)
// 			} else {
// 				s.Require().NoError(err)
// 				s.Require().Equal(tc.expectedBody, body)
// 			}
// 		})
// 	}
// }

func (s *ExecutorSuite) TestParseJSON() {
	testCases := []struct {
		name        string
		rawData     []byte
		expectError bool
		expectedMap map[string]any
	}{
		{
			name:        "Valid JSON object",
			rawData:     []byte(`{"key":"value"}`),
			expectError: false,
			expectedMap: map[string]any{"key": "value"},
		},
		{
			name:        "Valid JSON array of objects",
			rawData:     []byte(`[{"key":"value"}]`),
			expectError: false,
			expectedMap: map[string]any{"key": "value"},
		},
		{
			name:        "Invalid JSON",
			rawData:     []byte(`{"key":}`),
			expectError: true,
		},
		{
			name:        "JSON array with non-object element",
			rawData:     []byte(`[1, 2, 3]`),
			expectError: true,
		},
		{
			name:        "Empty JSON array",
			rawData:     []byte(`[]`),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			parsedData, err := parseJSON(tc.rawData)
			if tc.expectError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Require().Equal(tc.expectedMap, parsedData)
			}
		})
	}
}

func (s *ExecutorSuite) TestExtractDataByPath() {
	data := map[string]any{
		"level1": map[string]any{
			"level2": "value",
		},
		"items": []any{
			map[string]any{"name": "item1"},
			map[string]any{"name": "item2"},
		},
	}

	testCases := []struct {
		name          string
		path          string
		expectError   bool
		expectedValue string
	}{
		{
			name:          "Nested object path",
			path:          "level1.level2",
			expectError:   false,
			expectedValue: "value",
		},
		{
			name:          "Array path",
			path:          "items.1.name",
			expectError:   false,
			expectedValue: "item2",
		},
		{
			name:        "Invalid path - key not found",
			path:        "level1.nonexistent",
			expectError: true,
		},
		{
			name:        "Invalid path - out of bounds",
			path:        "items.2.name",
			expectError: true,
		},
		{
			name:        "Empty path",
			path:        "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			value, err := extractDataByPath(data, tc.path)
			if tc.expectError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Require().Equal(tc.expectedValue, value)
			}
		})
	}
}

func (s *ExecutorSuite) TestParseArrayIndex() {
	testCases := []struct {
		name        string
		input       string
		expectError bool
		expectedInt int
	}{
		{
			name:        "Valid index",
			input:       "123",
			expectError: false,
			expectedInt: 123,
		},
		{
			name:        "Invalid character",
			input:       "1a3",
			expectError: true,
		},
		{
			name:        "Empty string",
			input:       "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			index, err := parseArrayIndex(tc.input)
			if tc.expectError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Require().Equal(tc.expectedInt, index)
			}
		})
	}
}
