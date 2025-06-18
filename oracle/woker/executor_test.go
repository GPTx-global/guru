package woker

import (
	"fmt"
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
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json":
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"price":{"usd":50000.123},"rates":{"KRW":1300.5},"data":{"tags":["price","crypto"],"points":[{"x":1,"y":2},{"x":3,"y":4}]}}`)
		case "/json-array":
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `[{"name":"John","age":30},{"name":"Jane","age":25}]`)
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "Internal Server Error")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func (suite *ExecutorTestSuite) TearDownTest() {
	suite.server.Close()
}

func (suite *ExecutorTestSuite) TestExecuteJob_Success() {
	job := &types.Job{
		ID:    1,
		URL:   suite.server.URL + "/json",
		Path:  "price.usd",
		Nonce: 1,
		Delay: 0,
	}
	result := executeJob(job)
	suite.Require().NotNil(result)
	suite.Equal(uint64(1), result.ID)
	suite.Equal("50000.123", result.Data)
}

func (suite *ExecutorTestSuite) TestExecuteJob_WithDelay() {
	job := &types.Job{
		ID:    2,
		URL:   suite.server.URL + "/json",
		Path:  "price.usd",
		Nonce: 2, // Nonce > 1 triggers delay
		Delay: 50 * time.Millisecond,
	}

	start := time.Now()
	result := executeJob(job)
	duration := time.Since(start)

	suite.NotNil(result)
	suite.GreaterOrEqual(duration, 50*time.Millisecond)
}

func (suite *ExecutorTestSuite) TestExecuteJob_InvalidURL() {
	job := &types.Job{ID: 3, URL: "http://invalid-url", Path: "data"}
	result := executeJob(job)
	suite.Nil(result)
}

func (suite *ExecutorTestSuite) TestParseJSON() {
	testCases := []struct {
		name     string
		rawData  []byte
		hasError bool
	}{
		{"valid object", []byte(`{"a":1}`), false},
		{"valid array of objects", []byte(`[{"a":1}]`), false},
		{"invalid json", []byte(`[{"a":1]`), true},
		{"array of primitives", []byte(`[1,2,3]`), true},
		{"empty array", []byte(`[]`), true},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			_, err := parseJSON(tc.rawData)
			if tc.hasError {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
		})
	}
}

func (suite *ExecutorTestSuite) TestExtractDataByPath() {
	testCases := []struct {
		name     string
		path     string
		expected string
		hasError bool
	}{
		{"simple path", "price.usd", "50000.123", false},
		{"nested array path", "data.points.1.y", "4", false},
		{"invalid path", "price.eur", "", true},
		{"empty path", "", "", true},
		{"path through non-object", "price.usd.extra", "", true},
	}

	data, err := parseJSON([]byte(`{"price":{"usd":50000.123},"rates":{"KRW":1300.5},"data":{"tags":["price","crypto"],"points":[{"x":1,"y":2},{"x":3,"y":4}]}}`))
	suite.Require().NoError(err)

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			val, err := extractDataByPath(data, tc.path)
			if tc.hasError {
				suite.Error(err)
			} else {
				suite.NoError(err)
				suite.Equal(tc.expected, val)
			}
		})
	}
}
