package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/GPTx-global/guru/oracled/types"
)

type Executor struct {
	client *http.Client
	ctx    context.Context
}

func NewExecutor(ctx context.Context) *Executor {
	fmt.Printf("[ START ] NewExecutor\n")

	e := &Executor{
		client: &http.Client{},
		ctx:    ctx,
	}

	fmt.Printf("[  END  ] NewExecutor: SUCCESS\n")
	return e
}

func (e *Executor) ExecuteJob(job *types.Job) (*types.OracleData, error) {
	fmt.Printf("[ START ] ExecuteJob - ID: %d, Nonce: %d\n", job.ID, job.Nonce)

	if 1 < job.Nonce {
		time.Sleep(job.Delay)
	}

	rawData, err := e.fetchRawData(job.URL)
	if err != nil {
		fmt.Printf("[  END  ] ExecuteJob: ERROR - failed to fetch raw data for job %d: %v\n", job.ID, err)
		return nil, fmt.Errorf("failed to fetch raw data for job %d: %w", job.ID, err)
	}

	if err := e.validateResponse(rawData); err != nil {
		fmt.Printf("[  END  ] ExecuteJob: ERROR - invalid response for job %d: %v\n", job.ID, err)
		return nil, fmt.Errorf("invalid response for job %d: %w", job.ID, err)
	}

	parsedData, err := e.parseJSON(rawData)
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

	resp, err := e.client.Do(req)
	if err != nil {
		fmt.Printf("[  END  ] fetchRawData: ERROR - failed to do request: %v\n", err)
		return nil, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[  END  ] fetchRawData: ERROR - HTTP %d: %s\n",
			resp.StatusCode, resp.Status)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
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

	// Basic JSON validation
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
		fmt.Printf("[  END  ] parseJSON: ERROR - failed to unmarshal JSON: %v\n", err)
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
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

	// Split path by dots to navigate nested objects
	pathParts := strings.Split(path, ".")

	current := interface{}(data)
	for _, part := range pathParts {

		switch v := current.(type) {
		case map[string]interface{}:
			if val, exists := v[part]; exists {
				current = val
			} else {
				fmt.Printf("[  END  ] extractDataByPath: ERROR - key '%s' not found\n", part)
				return "", fmt.Errorf("key '%s' not found", part)
			}
		default:
			fmt.Printf("[  END  ] extractDataByPath: ERROR - invalid path at '%s'\n", part)
			return "", fmt.Errorf("invalid path at '%s'", part)
		}
	}

	// Convert final value to string
	result := fmt.Sprintf("%v", current)
	fmt.Printf("[SUCCESS] extractDataByPath: Extracted value: %s\n", result)
	fmt.Printf("[  END  ] extractDataByPath: SUCCESS\n")
	return result, nil
}
