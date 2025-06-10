package retry

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RetryConfig 재시도 설정
type RetryConfig struct {
	MaxAttempts int           // 최대 재시도 횟수
	BaseDelay   time.Duration // 기본 딜레이
	MaxDelay    time.Duration // 최대 딜레이
	Multiplier  float64       // 백오프 승수
}

// DefaultRetryConfig 기본 재시도 설정
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
	}
}

// NetworkRetryConfig 네트워크 관련 재시도 설정
func NetworkRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts: 10,
		BaseDelay:   2 * time.Second,
		MaxDelay:    60 * time.Second,
		Multiplier:  1.5,
	}
}

// TransactionRetryConfig 트랜잭션 관련 재시도 설정
func TransactionRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		Multiplier:  2.0,
	}
}

// RetryableFunc 재시도 가능한 함수 타입
type RetryableFunc func() error

// IsRetryable 재시도 가능한 에러인지 판단하는 함수 타입
type IsRetryable func(error) bool

// DefaultIsRetryable 기본 재시도 가능 에러 판단
func DefaultIsRetryable(err error) bool {
	if err == nil {
		return false
	}
	// 네트워크 관련 에러, 일시적 에러 등을 재시도 가능으로 판단
	errStr := err.Error()
	retryableErrors := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"network is unreachable",
		"no such host",
		"connection reset",
		"broken pipe",
		"i/o timeout",
		"context deadline exceeded",
		"account sequence mismatch",
		"insufficient funds",
	}

	for _, retryableErr := range retryableErrors {
		if contains(errStr, retryableErr) {
			return true
		}
	}
	return false
}

// TransactionIsRetryable 트랜잭션 전용 재시도 가능 에러 판단
func TransactionIsRetryable(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	retryableErrors := []string{
		"account sequence mismatch",
		"invalid sequence",
		"connection refused",
		"timeout",
		"network is unreachable",
		"tx already exists in cache",
	}

	for _, retryableErr := range retryableErrors {
		if contains(errStr, retryableErr) {
			return true
		}
	}
	return false
}

// Do 재시도 로직 실행
func Do(ctx context.Context, config *RetryConfig, fn RetryableFunc, isRetryable IsRetryable) error {
	fmt.Printf("[ START ] retry.Do - MaxAttempts: %d\n", config.MaxAttempts)

	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			fmt.Printf("[  END  ] retry.Do: CANCELLED - context cancelled\n")
			return ctx.Err()
		default:
		}

		fmt.Printf("[ RETRY ] Attempt %d/%d\n", attempt, config.MaxAttempts)

		err := fn()
		if err == nil {
			fmt.Printf("[  END  ] retry.Do: SUCCESS - completed on attempt %d\n", attempt)
			return nil
		}

		lastErr = err
		fmt.Printf("[  WARN ] retry.Do: Attempt %d failed - %v\n", attempt, err)

		// 마지막 시도라면 재시도하지 않음
		if attempt == config.MaxAttempts {
			break
		}

		// 재시도 가능한 에러인지 확인
		if !isRetryable(err) {
			fmt.Printf("[  END  ] retry.Do: ERROR - non-retryable error: %v\n", err)
			return err
		}

		// 백오프 딜레이 계산
		delay := calculateDelay(config, attempt)
		fmt.Printf("[ DELAY ] Waiting %v before next attempt\n", delay)

		select {
		case <-ctx.Done():
			fmt.Printf("[  END  ] retry.Do: CANCELLED - context cancelled during delay\n")
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	fmt.Printf("[  END  ] retry.Do: ERROR - all attempts exhausted: %v\n", lastErr)
	return fmt.Errorf("all %d attempts failed, last error: %w", config.MaxAttempts, lastErr)
}

// calculateDelay 백오프 딜레이 계산
func calculateDelay(config *RetryConfig, attempt int) time.Duration {
	delay := float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(attempt-1))

	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	return time.Duration(delay)
}

// CircuitBreaker 회로 차단기
type CircuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration
	failures     int
	lastFailTime time.Time
	state        CircuitState
}

// CircuitState 회로 상태
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// NewCircuitBreaker 새로운 회로 차단기 생성
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        StateClosed,
	}
}

// Execute 회로 차단기로 함수 실행
func (cb *CircuitBreaker) Execute(fn RetryableFunc) error {
	fmt.Printf("[ START ] CircuitBreaker.Execute - State: %v, Failures: %d\n", cb.state, cb.failures)

	if cb.state == StateOpen {
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
			fmt.Printf("[ STATE ] CircuitBreaker: Open -> HalfOpen\n")
		} else {
			fmt.Printf("[  END  ] CircuitBreaker.Execute: BLOCKED - circuit is open\n")
			return fmt.Errorf("circuit breaker is open")
		}
	}

	err := fn()

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		if cb.failures >= cb.maxFailures {
			cb.state = StateOpen
			fmt.Printf("[ STATE ] CircuitBreaker: -> Open (failures: %d)\n", cb.failures)
		}

		fmt.Printf("[  END  ] CircuitBreaker.Execute: ERROR - %v\n", err)
		return err
	}

	// 성공 시 상태 리셋
	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		fmt.Printf("[ STATE ] CircuitBreaker: HalfOpen -> Closed\n")
	}
	cb.failures = 0

	fmt.Printf("[  END  ] CircuitBreaker.Execute: SUCCESS\n")
	return nil
}

// GetState 현재 회로 상태 반환
func (cb *CircuitBreaker) GetState() CircuitState {
	return cb.state
}

// contains 문자열 포함 여부 확인
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
