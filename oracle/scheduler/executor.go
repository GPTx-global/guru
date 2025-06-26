package scheduler

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
	once       sync.Once    // Ensures HTTP client is initialized only once
	httpClient *http.Client // Singleton HTTP client for optimal connection reuse
)

// executorClient returns a singleton HTTP client optimized for oracle data fetching.
// It uses sync.Once to ensure the client is created only once and configured
// with appropriate timeouts and connection pooling for high-performance operation.
func executorClient() *http.Client {
	once.Do(func() {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        1000,             // Maximum number of idle connections
				MaxIdleConnsPerHost: 100,              // Maximum idle connections per host
				IdleConnTimeout:     90 * time.Second, // How long idle connections stay open
				MaxConnsPerHost:     200,              // Maximum connections per host
				WriteBufferSize:     32 * 1024,        // Write buffer size for performance
				ReadBufferSize:      32 * 1024,        // Read buffer size for performance
			},
		}
	})

	return httpClient
}

// executeJob processes a job by fetching data from the specified URL and extracting
// the value using the provided path. It handles delays for subsequent executions
// and returns a JobResult containing the extracted data.
func executeJob(job types.Job) (types.JobResult, error) {
	// Apply delay for subsequent executions (nonce > 1)
	if 1 < job.Nonce {
		<-time.After(job.Delay)
	}

	// Fetch raw data from the URL
	rawData, err := fetchRawData(job.URL)
	if err != nil {
		return types.JobResult{}, fmt.Errorf("failed to fetch raw data: %w", err)
	}

	// Parse the JSON response
	var parsedData map[string]any
	parsedData, err = parseJSON(rawData)
	if err != nil {
		return types.JobResult{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Extract the specific value using the path
	extractedValue, err := extractDataByPath(parsedData, job.Path)
	if err != nil {
		return types.JobResult{}, fmt.Errorf("failed to extract data by path: %w", err)
	}

	// Log the execution for debugging
	fmt.Printf("%2d/%-5d: %s\n", job.ID, job.Nonce, extractedValue)

	// Create and return the job result
	jobResult := types.JobResult{
		ID:    job.ID,
		Data:  extractedValue,
		Nonce: job.Nonce,
	}

	return jobResult, nil
}

// fetchRawData makes an HTTP GET request to the specified URL and returns the response body.
// It implements a retry mechanism for transient errors and handles various HTTP status codes.
// The function retries indefinitely for network errors and 5xx server errors.
func fetchRawData(url string) ([]byte, error) {
	var attemptCount uint64
	for attemptCount = 0; ; attemptCount++ {
		// Add delay for retry attempts
		if 0 < attemptCount {
			retryDelay := time.Second // Exponential backoff: 1s, 2s
			fmt.Printf("Request to %s failed. Retrying in %v... (Attempt %d)\n", url, retryDelay, attemptCount+1)
			time.Sleep(retryDelay)
		}

		// Create HTTP request with proper headers
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP request: %w", err) // Non-retryable error
		}

		// Set standard headers for oracle requests
		req.Header.Set("User-Agent", "Oracle-Daemon/1.0")
		req.Header.Set("Accept", "application/json")

		// Execute the HTTP request
		res, err := executorClient().Do(req)
		if err != nil {
			continue // Retry on network-level errors
		}

		// Handle successful response
		if res.StatusCode == http.StatusOK {
			body, readErr := io.ReadAll(res.Body)
			res.Body.Close()
			if readErr != nil {
				// This is unlikely but if it happens, it's better to not retry
				return nil, fmt.Errorf("failed to read response body: %w", readErr)
			}
			return body, nil // Success
		}

		// Retry on 5xx server errors (temporary server issues)
		if res.StatusCode >= 500 {
			res.Body.Close()
			continue
		}

		// For any other non-OK status, fail immediately (e.g., 4xx client errors)
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		return nil, fmt.Errorf("unexpected HTTP status: %s (%s)", res.Status, string(body))
	}
}

// parseJSON parses raw JSON data into a map structure.
// It handles both JSON objects and arrays, extracting the first element for arrays.
// Returns an error if the JSON is invalid or if an array is empty.
func parseJSON(rawData []byte) (map[string]any, error) {
	var result map[string]any

	// Try to parse as JSON object first
	if err := json.Unmarshal(rawData, &result); err != nil {
		// If object parsing fails, try as generic JSON (could be array)
		var anyResult any
		if unmarshalErr := json.Unmarshal(rawData, &anyResult); unmarshalErr != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		// Handle array case by extracting first element
		if arr, ok := anyResult.([]any); ok && len(arr) > 0 {
			if obj, ok := arr[0].(map[string]any); ok {
				result = obj
			} else {
				return nil, fmt.Errorf("first array element is not a JSON object")
			}
		} else {
			return nil, fmt.Errorf("response is not a JSON object or array")
		}
	}

	return result, nil
}

// extractDataByPath extracts a value from nested JSON data using dot notation path.
// It supports traversing objects and arrays, with numeric indices for array access.
// Returns the extracted value as a string or an error if the path is invalid.
func extractDataByPath(data map[string]any, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Split the path by dots to get individual components
	pathParts := strings.Split(path, ".")

	current := any(data)
	for _, part := range pathParts {
		switch v := current.(type) {
		case map[string]any:
			// Navigate through object properties
			if val, exists := v[part]; exists {
				current = val
			} else {
				return "", fmt.Errorf("key '%s' not found in path %s", part, path)
			}
		case []any:
			// Navigate through array indices
			if index, parseErr := parseArrayIndex(part); parseErr == nil {
				if index >= 0 && index < len(v) {
					current = v[index]
				} else {
					return "", fmt.Errorf("array index %d out of bounds (length: %d)", index, len(v))
				}
			} else {
				return "", fmt.Errorf("invalid array index '%s': %v", part, parseErr)
			}
		default:
			return "", fmt.Errorf("cannot traverse '%s' in type %T", part, current)
		}
	}

	// Convert final value to string representation
	return fmt.Sprintf("%v", current), nil
}

// parseArrayIndex converts a string to an integer for array indexing.
// It manually parses the string to avoid importing strconv and provides
// better error messages for invalid indices.
func parseArrayIndex(s string) (int, error) {
	if s == "" {
		return -1, fmt.Errorf("array index cannot be empty")
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
