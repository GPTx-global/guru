package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/GPTx-global/guru/oracled/retry"
	"github.com/GPTx-global/guru/oracled/types"
)

type Executor struct {
	client             *http.Client
	ctx                context.Context
	httpCircuitBreaker *retry.CircuitBreaker
}

func NewExecutor(ctx context.Context) *Executor {
	fmt.Printf("[ START ] NewExecutor\n")

	e := &Executor{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		ctx:                ctx,
		httpCircuitBreaker: retry.NewCircuitBreaker(5, 2*time.Minute),
	}

	fmt.Printf("[  END  ] NewExecutor: SUCCESS\n")
	return e
}

func (e *Executor) ExecuteJob(job *types.Job) (*types.OracleData, error) {
	fmt.Printf("[ START ] ExecuteJob - ID: %d, Nonce: %d\n", job.ID, job.Nonce)

	if 1 < job.Nonce {
		fmt.Printf("[ DELAY ] ExecuteJob: Waiting %v for nonce %d\n", job.Delay, job.Nonce)
		select {
		case <-time.After(job.Delay):
		case <-e.ctx.Done():
			return nil, fmt.Errorf("context cancelled during delay")
		}
	}

	var rawData []byte
	err := retry.Do(e.ctx, retry.NetworkRetryConfig(),
		func() error {
			return e.httpCircuitBreaker.Execute(func() error {
				var err error
				rawData, err = e.fetchRawData(job.URL)
				return err
			})
		},
		func(err error) bool {
			return e.isRetryableHTTPError(err)
		},
	)

	if err != nil {
		fmt.Printf("[  END  ] ExecuteJob: ERROR - failed to fetch raw data for job %d: %v\n", job.ID, err)
		return nil, fmt.Errorf("failed to fetch raw data for job %d: %w", job.ID, err)
	}

	if err := e.validateResponse(rawData); err != nil {
		fmt.Printf("[  END  ] ExecuteJob: ERROR - invalid response for job %d: %v\n", job.ID, err)
		return nil, fmt.Errorf("invalid response for job %d: %w", job.ID, err)
	}

	var parsedData map[string]interface{}
	err = retry.Do(e.ctx, retry.DefaultRetryConfig(),
		func() error {
			var err error
			parsedData, err = e.parseJSON(rawData)
			return err
		},
		func(err error) bool {
			return false
		},
	)

	if err != nil {
		fmt.Printf("[  END  ] ExecuteJob: ERROR - failed to parse JSON for job %d: %v\n", job.ID, err)
		return nil, fmt.Errorf("failed to parse JSON for job %d: %w", job.ID, err)
	}

	extractedValue, err := e.extractDataByPath(parsedData, job.Path)
	if err != nil {
		fmt.Printf("[  END  ] ExecuteJob: ERROR - failed to extract data for job %d: %v\n", job.ID, err)
		return nil, fmt.Errorf("failed to extract data for job %d: %w", job.ID, err)
	}

	oracleData := &types.OracleData{
		RequestID: job.ID,
		Data:      extractedValue,
		Nonce:     job.Nonce,
	}

	fmt.Printf("[SUCCESS] ExecuteJob: Oracle data created - RequestID: %d, Data: %s, Nonce: %d\n",
		oracleData.RequestID, oracleData.Data, oracleData.Nonce)
	fmt.Printf("[  END  ] ExecuteJob: SUCCESS\n")
	return oracleData, nil
}

func (e *Executor) fetchRawData(url string) ([]byte, error) {
	fmt.Printf("[ START ] fetchRawData\n")

	req, err := http.NewRequestWithContext(e.ctx, "GET", url, nil)
	if err != nil {
		fmt.Printf("[  END  ] fetchRawData: ERROR - failed to create request: %v\n", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Oracle-Daemon/1.0")
	req.Header.Set("Accept", "application/json")
	// Accept-Encoding 헤더를 제거하여 Go의 자동 gzip 압축 해제 기능 사용

	resp, err := e.client.Do(req)
	if err != nil {
		fmt.Printf("[  END  ] fetchRawData: ERROR - failed to do request: %v\n", err)
		return nil, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("[  END  ] fetchRawData: ERROR - HTTP %d: %s\n",
			resp.StatusCode, resp.Status)

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return nil, fmt.Errorf("client error HTTP %d: %s", resp.StatusCode, resp.Status)
		}
		return nil, fmt.Errorf("server error HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[  END  ] fetchRawData: ERROR - failed to read response body: %v\n", err)
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	fmt.Printf("[  END  ] fetchRawData: SUCCESS - %d bytes fetched\n", len(body))
	return body, nil
}

func (e *Executor) validateResponse(data []byte) error {
	fmt.Printf("[ START ] validateResponse - Size: %d bytes\n", len(data))

	if len(data) == 0 {
		fmt.Printf("[  END  ] validateResponse: ERROR - empty response\n")
		return fmt.Errorf("empty response")
	}

	maxSize := 10 * 1024 * 1024 // 10MB
	if len(data) > maxSize {
		fmt.Printf("[  END  ] validateResponse: ERROR - response too large: %d bytes\n", len(data))
		return fmt.Errorf("response too large: %d bytes (max: %d)", len(data), maxSize)
	}

	var temp interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		fmt.Printf("[  END  ] validateResponse: ERROR - invalid JSON: %v\n", err)
		return fmt.Errorf("invalid JSON: %w", err)
	}

	fmt.Printf("[  END  ] validateResponse: SUCCESS\n")
	return nil
}

func (e *Executor) parseJSON(data []byte) (map[string]interface{}, error) {
	fmt.Printf("[ START ] parseJSON\n")

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		var anyResult interface{}
		if unmarshalErr := json.Unmarshal(data, &anyResult); unmarshalErr != nil {
			fmt.Printf("[  END  ] parseJSON: ERROR - failed to unmarshal JSON: %v\n", err)
			return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		if arr, ok := anyResult.([]interface{}); ok && len(arr) > 0 {
			if obj, ok := arr[0].(map[string]interface{}); ok {
				result = obj
			} else {
				fmt.Printf("[  END  ] parseJSON: ERROR - first array element is not an object\n")
				return nil, fmt.Errorf("first array element is not an object")
			}
		} else {
			fmt.Printf("[  END  ] parseJSON: ERROR - response is not a JSON object or array\n")
			return nil, fmt.Errorf("response is not a JSON object or array")
		}
	}

	fmt.Printf("[  END  ] parseJSON: SUCCESS\n")
	return result, nil
}

func (e *Executor) extractDataByPath(data map[string]interface{}, path string) (string, error) {
	fmt.Printf("[ START ] extractDataByPath - Path: %s\n", path)

	if path == "" {
		fmt.Printf("[  END  ] extractDataByPath: ERROR - empty path\n")
		return "", fmt.Errorf("empty path")
	}

	pathParts := strings.Split(path, ".")

	current := interface{}(data)
	for i, part := range pathParts {
		fmt.Printf("[ PATH  ] extractDataByPath: Processing part %d: %s\n", i+1, part)

		switch v := current.(type) {
		case map[string]interface{}:
			if val, exists := v[part]; exists {
				current = val
				fmt.Printf("[ PATH  ] extractDataByPath: Found key '%s' with type %T\n", part, val)
			} else {
				fmt.Printf("[  END  ] extractDataByPath: ERROR - key '%s' not found\n", part)
				return "", fmt.Errorf("key '%s' not found in path %s", part, path)
			}
		case []interface{}:
			if index, parseErr := parseArrayIndex(part); parseErr == nil {
				if index >= 0 && index < len(v) {
					current = v[index]
					fmt.Printf("[ PATH  ] extractDataByPath: Accessed array index %d\n", index)
				} else {
					fmt.Printf("[  END  ] extractDataByPath: ERROR - array index %d out of bounds (length: %d)\n", index, len(v))
					return "", fmt.Errorf("array index %d out of bounds", index)
				}
			} else {
				fmt.Printf("[  END  ] extractDataByPath: ERROR - invalid array index '%s'\n", part)
				return "", fmt.Errorf("invalid array index '%s'", part)
			}
		default:
			fmt.Printf("[  END  ] extractDataByPath: ERROR - cannot traverse '%s' in type %T\n", part, current)
			return "", fmt.Errorf("cannot traverse '%s' in type %T", part, current)
		}
	}

	result := fmt.Sprintf("%v", current)
	fmt.Printf("[SUCCESS] extractDataByPath: Extracted value: %s\n", result)
	fmt.Printf("[  END  ] extractDataByPath: SUCCESS\n")
	return result, nil
}

func (e *Executor) isRetryableHTTPError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	retryableErrors := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"temporary failure",
		"server error HTTP 5",
		"no such host",
		"network is unreachable",
		"i/o timeout",
		"context deadline exceeded",
	}

	for _, retryableErr := range retryableErrors {
		if strings.Contains(errStr, retryableErr) {
			return true
		}
	}

	return false
}

func parseArrayIndex(s string) (int, error) {
	if s == "" {
		return -1, fmt.Errorf("empty string")
	}

	result := 0
	for _, char := range s {
		if char < '0' || char > '9' {
			return -1, fmt.Errorf("invalid character: %c", char)
		}
		result = result*10 + int(char-'0')
	}

	return result, nil
}

func (e *Executor) GetHealthCheckFunc() func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if e.client == nil {
			return fmt.Errorf("http client is nil")
		}

		if e.httpCircuitBreaker.GetState() == retry.StateOpen {
			return fmt.Errorf("http circuit breaker is open")
		}

		return nil
	}
}
