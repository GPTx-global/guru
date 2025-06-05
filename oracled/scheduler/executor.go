package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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

	// 점(.)으로 경로를 분할하여 중첩된 구조 탐색
	pathParts := strings.Split(path, ".")

	// 현재 데이터를 interface{}로 시작
	var currentData interface{} = data

	// 각 경로 부분을 순차적으로 탐색
	for i, part := range pathParts {
		switch current := currentData.(type) {
		case map[string]interface{}:
			// 맵에서 키 찾기
			value, exists := current[part]
			if !exists {
				return "", fmt.Errorf("path '%s' not found at level %d (key: '%s')", path, i, part)
			}
			currentData = value

		case []interface{}:
			// 배열 인덱스 처리 (숫자인 경우)
			index, err := strconv.Atoi(part)
			if err != nil {
				return "", fmt.Errorf("path '%s' at level %d: expected array index but got '%s'", path, i, part)
			}
			if index < 0 || index >= len(current) {
				return "", fmt.Errorf("path '%s' at level %d: array index %d out of bounds (length: %d)", path, i, index, len(current))
			}
			currentData = current[index]

		default:
			// 더 이상 탐색할 수 없는 타입
			return "", fmt.Errorf("path '%s' at level %d: cannot traverse further, current type: %T", path, i, current)
		}
	}

	// 최종 값을 문자열로 변환
	return e.convertToString(currentData)
}

// convertToString은 다양한 타입의 값을 문자열로 변환합니다
func (e *Executor) convertToString(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case float64:
		return fmt.Sprintf("%.8f", v), nil
	case float32:
		return fmt.Sprintf("%.8f", v), nil
	case int:
		return fmt.Sprintf("%d", v), nil
	case int64:
		return fmt.Sprintf("%d", v), nil
	case int32:
		return fmt.Sprintf("%d", v), nil
	case bool:
		return fmt.Sprintf("%t", v), nil
	case nil:
		return "", nil
	default:
		// 복잡한 객체는 JSON으로 직렬화
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to convert value to string: %w", err)
		}
		return string(jsonBytes), nil
	}
}
