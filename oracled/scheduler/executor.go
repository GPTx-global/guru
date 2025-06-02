package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/GPTx-global/guru/oracled/types"
)

type Executor struct {
	client *http.Client
	ctx    context.Context
}

func NewExecutor(ctx context.Context) *Executor {
	return &Executor{
		client: &http.Client{},
		ctx:    ctx,
	}
}

func (e *Executor) ExecuteJob(job *types.Job) (*types.OracleData, error) {
	if 0 < job.Nonce {
		time.Sleep(job.Delay)
	}

	rawData, err := e.fetchRawData(job.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch raw data for job %s: %w", job.ID, err)
	}

	if err := e.validateResponse(rawData); err != nil {
		return nil, fmt.Errorf("invalid response for job %s: %w", job.ID, err)
	}

	parsedData, err := e.parseJSON(rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON for job %s: %w", job.ID, err)
	}

	extractedValue, err := e.extractDataByPath(parsedData, job.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to extract data for job %s: %w", job.ID, err)
	}

	oracleData := &types.OracleData{
		RequestID: job.ID,
		Data:      extractedValue,
		Nonce:     job.Nonce,
	}

	return oracleData, nil
}

func (e *Executor) fetchRawData(url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(e.ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Oracle-Daemon/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

func (e *Executor) validateResponse(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty response body")
	}

	var temp interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	return nil
}

func (e *Executor) parseJSON(data []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return result, nil
}

func (e *Executor) extractDataByPath(data map[string]interface{}, path string) (string, error) {
	if path == "" {
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return "", err
		}
		return string(jsonBytes), nil
	}

	// TODO: 더 복잡한 path 처리 필요
	if value, ok := data[path]; ok {
		switch v := value.(type) {
		case string:
			return v, nil
		case float64:
			return fmt.Sprintf("%.8f", v), nil
		case int64:
			return fmt.Sprintf("%d", v), nil
		default:
			return fmt.Sprintf("%v", v), nil
		}
	}

	return "", fmt.Errorf("path '%s' not found in data", path)
}
