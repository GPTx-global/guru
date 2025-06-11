package scheduler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/GPTx-global/guru/oracled/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutor(t *testing.T) {
	ctx := context.Background()
	executor := NewExecutor(ctx)

	require.NotNil(t, executor)
	assert.NotNil(t, executor.client)
	assert.Equal(t, ctx, executor.ctx)
	assert.NotNil(t, executor.httpCircuitBreaker)

	// HTTP 클라이언트 설정 확인
	assert.Equal(t, 30*time.Second, executor.client.Timeout)
}

func TestExecutor_ValidateResponse(t *testing.T) {
	executor := NewExecutor(context.Background())

	testCases := []struct {
		name        string
		data        []byte
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid JSON",
			data:        []byte(`{"price": 100, "symbol": "BTC"}`),
			expectError: false,
		},
		{
			name:        "empty response",
			data:        []byte{},
			expectError: true,
			errorMsg:    "empty response",
		},
		{
			name:        "invalid JSON",
			data:        []byte(`{"price": 100, "symbol": }`),
			expectError: true,
			errorMsg:    "invalid JSON",
		},
		{
			name:        "too large response",
			data:        make([]byte, 11*1024*1024), // 11MB
			expectError: true,
			errorMsg:    "response too large",
		},
		{
			name:        "valid array JSON",
			data:        []byte(`[{"price": 100}, {"price": 200}]`),
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := executor.validateResponse(tc.data)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExecutor_ParseJSON(t *testing.T) {
	executor := NewExecutor(context.Background())

	testCases := []struct {
		name        string
		data        []byte
		expectError bool
		expectedLen int
	}{
		{
			name:        "simple object",
			data:        []byte(`{"price": 100, "symbol": "BTC"}`),
			expectError: false,
			expectedLen: 2,
		},
		{
			name:        "array with objects",
			data:        []byte(`[{"price": 100}, {"price": 200}]`),
			expectError: false,
			expectedLen: 1, // 첫 번째 요소만 반환
		},
		{
			name:        "empty object",
			data:        []byte(`{}`),
			expectError: false,
			expectedLen: 0,
		},
		{
			name:        "invalid JSON",
			data:        []byte(`{"price": }`),
			expectError: true,
		},
		{
			name:        "array with non-object",
			data:        []byte(`[1, 2, 3]`),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := executor.parseJSON(tc.data)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tc.expectedLen)
			}
		})
	}
}

func TestExecutor_ExtractDataByPath(t *testing.T) {
	executor := NewExecutor(context.Background())

	data := map[string]interface{}{
		"price": 42000.5,
		"data": map[string]interface{}{
			"usd": 42000.5,
			"eur": 35000.0,
		},
		"meta": map[string]interface{}{
			"timestamp": 1234567890,
			"source":    "exchange",
		},
		"array": []interface{}{
			map[string]interface{}{"value": 100},
			map[string]interface{}{"value": 200},
		},
	}

	testCases := []struct {
		name         string
		path         string
		expectedData string
		expectError  bool
	}{
		{
			name:         "simple field",
			path:         "price",
			expectedData: "42000.5",
			expectError:  false,
		},
		{
			name:         "nested field",
			path:         "data.usd",
			expectedData: "42000.5",
			expectError:  false,
		},
		{
			name:         "double nested field",
			path:         "meta.timestamp",
			expectedData: "1.23456789e+09",
			expectError:  false,
		},
		{
			name:         "array index access",
			path:         "array[0].value",
			expectedData: "100",
			expectError:  false,
		},
		{
			name:         "array second element",
			path:         "array[1].value",
			expectedData: "200",
			expectError:  false,
		},
		{
			name:        "non-existent field",
			path:        "nonexistent",
			expectError: true,
		},
		{
			name:        "invalid array index",
			path:        "array[5].value",
			expectError: true,
		},
		{
			name:        "invalid path format",
			path:        "data..usd",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := executor.extractDataByPath(data, tc.path)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedData, result)
			}
		})
	}
}

func TestExecutor_FetchRawData_Success(t *testing.T) {
	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 요청 헤더 확인
		assert.Equal(t, "Oracle-Daemon/1.0", r.Header.Get("User-Agent"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"price": 42000, "symbol": "BTC"}`))
	}))
	defer server.Close()

	executor := NewExecutor(context.Background())
	data, err := executor.fetchRawData(server.URL)

	require.NoError(t, err)
	assert.JSONEq(t, `{"price": 42000, "symbol": "BTC"}`, string(data))
}

func TestExecutor_FetchRawData_HTTPErrors(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		expectErr  bool
	}{
		{
			name:       "success 200",
			statusCode: http.StatusOK,
			expectErr:  false,
		},
		{
			name:       "client error 404",
			statusCode: http.StatusNotFound,
			expectErr:  true,
		},
		{
			name:       "server error 500",
			statusCode: http.StatusInternalServerError,
			expectErr:  true,
		},
		{
			name:       "redirect 302",
			statusCode: http.StatusFound,
			expectErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(`{"error": "test error"}`))
			}))
			defer server.Close()

			executor := NewExecutor(context.Background())
			_, err := executor.fetchRawData(server.URL)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExecutor_ExecuteJob_WithDelay(t *testing.T) {
	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"price": 42000}}`))
	}))
	defer server.Close()

	executor := NewExecutor(context.Background())

	job := &types.Job{
		ID:    123,
		URL:   server.URL,
		Path:  "data.price",
		Nonce: 2, // Nonce > 1이므로 지연 발생
		Delay: 100 * time.Millisecond,
	}

	start := time.Now()
	result, err := executor.ExecuteJob(job)
	duration := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, result)

	// 지연이 발생했는지 확인 (약간의 여유를 둠)
	assert.True(t, duration >= 90*time.Millisecond)

	assert.Equal(t, uint64(123), result.RequestID)
	assert.Equal(t, "42000", result.Data)
	assert.Equal(t, uint64(2), result.Nonce)
}

func TestExecutor_ExecuteJob_NoDelay(t *testing.T) {
	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"price": 42000}}`))
	}))
	defer server.Close()

	executor := NewExecutor(context.Background())

	job := &types.Job{
		ID:    123,
		URL:   server.URL,
		Path:  "data.price",
		Nonce: 1, // Nonce <= 1이므로 지연 없음
		Delay: 1 * time.Second,
	}

	start := time.Now()
	result, err := executor.ExecuteJob(job)
	duration := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, result)

	// 지연이 발생하지 않았는지 확인
	assert.True(t, duration < 100*time.Millisecond)

	assert.Equal(t, uint64(123), result.RequestID)
	assert.Equal(t, "42000", result.Data)
	assert.Equal(t, uint64(1), result.Nonce)
}

func TestExecutor_IsRetryableHTTPError(t *testing.T) {
	executor := NewExecutor(context.Background())

	testCases := []struct {
		name        string
		err         error
		expectRetry bool
	}{
		{
			name:        "timeout error",
			err:         fmt.Errorf("timeout"),
			expectRetry: true,
		},
		{
			name:        "connection error",
			err:         fmt.Errorf("connection refused"),
			expectRetry: true,
		},
		{
			name:        "temporary error",
			err:         fmt.Errorf("temporary failure"),
			expectRetry: true,
		},
		{
			name:        "client error",
			err:         fmt.Errorf("client error HTTP 400"),
			expectRetry: false,
		},
		{
			name:        "not found error",
			err:         fmt.Errorf("not found"),
			expectRetry: false,
		},
		{
			name:        "server error",
			err:         fmt.Errorf("server error HTTP 500"),
			expectRetry: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := executor.isRetryableHTTPError(tc.err)
			assert.Equal(t, tc.expectRetry, result)
		})
	}
}

func TestExecutor_ExecuteJob_ContextCancellation(t *testing.T) {
	// 느린 서버 시뮬레이션
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // 2초 지연
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"price": 42000}}`))
	}))
	defer server.Close()

	// 짧은 타임아웃으로 context 생성
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	executor := NewExecutor(ctx)

	job := &types.Job{
		ID:    123,
		URL:   server.URL,
		Path:  "data.price",
		Nonce: 2,
		Delay: 50 * time.Millisecond,
	}

	_, err := executor.ExecuteJob(job)

	// Context cancellation으로 인한 에러 발생해야 함
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}
