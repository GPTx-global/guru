package scheduler

import (
	"context"
	"testing"
)

func TestExtractDataByPath(t *testing.T) {
	executor := NewExecutor(context.Background())

	// 테스트 데이터: 다중 레벨 중첩 구조
	testData := map[string]interface{}{
		"price": "42000.50",
		"data": map[string]interface{}{
			"amount":   "1000.25",
			"currency": "USD",
			"details": map[string]interface{}{
				"high": 45000.0,
				"low":  40000.0,
				"volume": map[string]interface{}{
					"btc": 123.45,
					"usd": 5000000.0,
				},
			},
		},
		"items": []interface{}{
			"first_item",
			map[string]interface{}{
				"name":  "second_item",
				"value": 999,
			},
			42.0,
		},
		"status":     true,
		"count":      100,
		"null_field": nil,
	}

	tests := []struct {
		name     string
		path     string
		expected string
		hasError bool
	}{
		// 단일 레벨 경로
		{
			name:     "Simple string field",
			path:     "price",
			expected: "42000.50",
			hasError: false,
		},
		{
			name:     "Integer field",
			path:     "count",
			expected: "100",
			hasError: false,
		},
		{
			name:     "Boolean field",
			path:     "status",
			expected: "true",
			hasError: false,
		},
		{
			name:     "Null field",
			path:     "null_field",
			expected: "",
			hasError: false,
		},

		// 2단계 중첩 경로
		{
			name:     "Nested string field",
			path:     "data.amount",
			expected: "1000.25",
			hasError: false,
		},
		{
			name:     "Nested currency field",
			path:     "data.currency",
			expected: "USD",
			hasError: false,
		},

		// 3단계 중첩 경로
		{
			name:     "Deep nested float field",
			path:     "data.details.high",
			expected: "45000.00000000",
			hasError: false,
		},
		{
			name:     "Very deep nested field",
			path:     "data.details.volume.btc",
			expected: "123.45000000",
			hasError: false,
		},
		{
			name:     "Very deep nested USD field",
			path:     "data.details.volume.usd",
			expected: "5000000.00000000",
			hasError: false,
		},

		// 배열 인덱스 경로
		{
			name:     "Array first element",
			path:     "items.0",
			expected: "first_item",
			hasError: false,
		},
		{
			name:     "Array nested object field",
			path:     "items.1.name",
			expected: "second_item",
			hasError: false,
		},
		{
			name:     "Array nested object value",
			path:     "items.1.value",
			expected: "999",
			hasError: false,
		},
		{
			name:     "Array float element",
			path:     "items.2",
			expected: "42.00000000",
			hasError: false,
		},

		// 에러 케이스
		{
			name:     "Non-existent field",
			path:     "nonexistent",
			expected: "",
			hasError: true,
		},
		{
			name:     "Non-existent nested field",
			path:     "data.nonexistent",
			expected: "",
			hasError: true,
		},
		{
			name:     "Array out of bounds",
			path:     "items.10",
			expected: "",
			hasError: true,
		},
		{
			name:     "Invalid array index",
			path:     "items.invalid",
			expected: "",
			hasError: true,
		},
		{
			name:     "Cannot traverse primitive",
			path:     "price.invalid",
			expected: "",
			hasError: true,
		},

		// 빈 경로 (전체 JSON 반환)
		{
			name:     "Empty path returns full JSON",
			path:     "",
			expected: "", // JSON 문자열이므로 정확한 비교는 생략
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.extractDataByPath(testData, tt.path)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// 빈 경로의 경우 JSON 문자열 확인
			if tt.path == "" {
				if result == "" {
					t.Errorf("Expected non-empty JSON string for empty path")
				}
				return
			}

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
