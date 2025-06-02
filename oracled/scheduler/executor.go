package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/GPTx-global/guru/oracled/types"
)

type Executor struct {
	client *http.Client
	ctx    context.Context
}

func NewExecutor() *Executor {
	return &Executor{
		client: &http.Client{},
		ctx:    context.Background(),
	}
}

func (e *Executor) ExecuteJob(job *types.Job) (*types.OracleData, error) {
	fmt.Printf("Executing job: %s (URL: %s, nonce: %d)\n", job.ID, job.URL, job.Nonce)

	// 외부 데이터 수집
	data, err := e.fetchExternalData(job)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data for job %s: %w", job.ID, err)
	}

	// Oracle 데이터 생성
	oracleData := &types.OracleData{
		RequestID: job.ID,    // Job.ID 그대로 사용 (nonce 포함하지 않음)
		Data:      data,      // 최근에 수집한 데이터
		Nonce:     job.Nonce, // Job.Nonce
	}

	fmt.Printf("Oracle data created for job %s (nonce: %d): %s\n", job.ID, job.Nonce, data)
	return oracleData, nil
}

func (e *Executor) fetchExternalData(job *types.Job) (string, error) {
	req, err := http.NewRequestWithContext(e.ctx, "GET", job.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Oracle-Daemon/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	// Path를 사용하여 데이터 추출
	value, err := e.extractDataByPath(result, job.Path)
	if err != nil {
		return "", fmt.Errorf("failed to extract data: %w", err)
	}

	return value, nil
}

func (e *Executor) extractDataByPath(data map[string]interface{}, path string) (string, error) {
	if path == "" {
		// Path가 없으면 전체 JSON을 문자열로 반환
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return "", err
		}
		return string(jsonBytes), nil
	}

	// 간단한 path 추출 (예: "price", "data.amount" 등)
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
