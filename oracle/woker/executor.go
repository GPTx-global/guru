package woker

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/GPTx-global/guru/oracle/types"
)

var (
	once       sync.Once
	httpClient *http.Client
)

// executorClient returns a singleton HTTP client optimized for oracle data fetching
func executorClient() *http.Client {
	once.Do(func() {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        1000,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
				MaxConnsPerHost:     200,
				WriteBufferSize:     32 * 1024,
				ReadBufferSize:      32 * 1024,
			},
		}
	})

	return httpClient
}

// executeJob processes a job by fetching data from URL and extracting value by path
func executeJob(job *types.Job) *types.JobResult {
	if 1 < job.Nonce {
		<-time.After(job.Delay)
	}

	rawData, err := fetchRawData(job.URL)
	if err != nil {
		fmt.Printf("failed to fetch raw data: %v\n", err)
		return nil
	}

	var parsedData map[string]any
	parsedData, err = parseJSON(rawData)
	if err != nil {
		fmt.Printf("failed to parse JSON: %v\n", err)
		return nil
	}

	extractedValue, err := extractDataByPath(parsedData, job.Path)
	if err != nil {
		fmt.Printf("failed to extract data by path: %v\n", err)
		return nil
	}

	jr := &types.JobResult{
		ID:    job.ID,
		Data:  extractedValue,
		Nonce: job.Nonce,
	}

	return jr
}

// fetchRawData makes HTTP GET request to the specified URL and returns response body
func fetchRawData(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Oracle-Daemon/1.0")
	req.Header.Set("Accept", "application/json")

	res, err := executorClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch raw data: %w", err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

// parseJSON parses raw JSON data into a map, handles both objects and arrays
func parseJSON(rawData []byte) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal(rawData, &result); err != nil {
		var anyResult any
		if unmarshalErr := json.Unmarshal(rawData, &anyResult); unmarshalErr != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		if arr, ok := anyResult.([]any); ok && len(arr) > 0 {
			if obj, ok := arr[0].(map[string]any); ok {
				result = obj
			} else {
				return nil, fmt.Errorf("first array element is not an object")
			}
		} else {
			return nil, fmt.Errorf("response is not a JSON object or array")
		}
	}

	return result, nil
}

// extractDataByPath extracts value from nested JSON data using dot notation path
func extractDataByPath(data map[string]any, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}

	pathParts := strings.Split(path, ".")

	current := any(data)
	for _, part := range pathParts {
		switch v := current.(type) {
		case map[string]any:
			if val, exists := v[part]; exists {
				current = val
			} else {
				return "", fmt.Errorf("key '%s' not found in path %s", part, path)
			}
		case []any:
			if index, parseErr := parseArrayIndex(part); parseErr == nil {
				if index >= 0 && index < len(v) {
					current = v[index]
				} else {
					return "", fmt.Errorf("array index %d out of bounds", index)
				}
			} else {
				return "", fmt.Errorf("invalid array index '%s'", part)
			}
		default:
			return "", fmt.Errorf("cannot traverse '%s' in type %T", part, current)
		}
	}

	return fmt.Sprintf("%v", current), nil
}

// parseArrayIndex converts string to integer for array indexing
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
