package health

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HealthCheck 헬스 체크 인터페이스
type HealthCheck interface {
	Check(ctx context.Context) error
	Name() string
}

// HealthChecker 헬스 체커
type HealthChecker struct {
	checks   map[string]HealthCheck
	mutex    sync.RWMutex
	interval time.Duration
	status   map[string]HealthStatus
}

// HealthStatus 헬스 상태
type HealthStatus struct {
	Healthy   bool
	LastCheck time.Time
	LastError error
}

// NewHealthChecker 새로운 헬스 체커 생성
func NewHealthChecker(interval time.Duration) *HealthChecker {
	return &HealthChecker{
		checks:   make(map[string]HealthCheck),
		status:   make(map[string]HealthStatus),
		interval: interval,
	}
}

// AddCheck 헬스 체크 추가
func (hc *HealthChecker) AddCheck(check HealthCheck) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	name := check.Name()
	hc.checks[name] = check
	hc.status[name] = HealthStatus{
		Healthy:   true,
		LastCheck: time.Now(),
		LastError: nil,
	}

	fmt.Printf("[ HEALTH] Added health check: %s\n", name)
}

// Start 헬스 체크 시작
func (hc *HealthChecker) Start(ctx context.Context) {
	fmt.Printf("[ START ] HealthChecker.Start - interval: %v\n", hc.interval)

	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	// 초기 체크 실행
	hc.runChecks(ctx)

	for {
		select {
		case <-ticker.C:
			hc.runChecks(ctx)
		case <-ctx.Done():
			fmt.Printf("[  END  ] HealthChecker.Start: context cancelled\n")
			return
		}
	}
}

// runChecks 모든 헬스 체크 실행
func (hc *HealthChecker) runChecks(ctx context.Context) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	for name, check := range hc.checks {
		go func(name string, check HealthCheck) {
			fmt.Printf("[ CHECK ] Running health check: %s\n", name)

			err := check.Check(ctx)

			hc.mutex.Lock()
			hc.status[name] = HealthStatus{
				Healthy:   err == nil,
				LastCheck: time.Now(),
				LastError: err,
			}
			hc.mutex.Unlock()

			if err != nil {
				fmt.Printf("[  WARN ] Health check failed - %s: %v\n", name, err)
			} else {
				fmt.Printf("[ HEALTH] Health check passed: %s\n", name)
			}
		}(name, check)
	}
}

// GetStatus 헬스 상태 조회
func (hc *HealthChecker) GetStatus() map[string]HealthStatus {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	result := make(map[string]HealthStatus)
	for name, status := range hc.status {
		result[name] = status
	}

	return result
}

// IsHealthy 전체 시스템 건강 상태 확인
func (hc *HealthChecker) IsHealthy() bool {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	for _, status := range hc.status {
		if !status.Healthy {
			return false
		}
	}

	return true
}

// RPCHealthCheck RPC 연결 헬스 체크
type RPCHealthCheck struct {
	name      string
	checkFunc func(ctx context.Context) error
}

// NewRPCHealthCheck RPC 헬스 체크 생성
func NewRPCHealthCheck(name string, checkFunc func(ctx context.Context) error) *RPCHealthCheck {
	return &RPCHealthCheck{
		name:      name,
		checkFunc: checkFunc,
	}
}

// Check 헬스 체크 실행
func (rhc *RPCHealthCheck) Check(ctx context.Context) error {
	return rhc.checkFunc(ctx)
}

// Name 헬스 체크 이름 반환
func (rhc *RPCHealthCheck) Name() string {
	return rhc.name
}

// KeyringHealthCheck 키링 헬스 체크
type KeyringHealthCheck struct {
	name      string
	checkFunc func(ctx context.Context) error
}

// NewKeyringHealthCheck 키링 헬스 체크 생성
func NewKeyringHealthCheck(name string, checkFunc func(ctx context.Context) error) *KeyringHealthCheck {
	return &KeyringHealthCheck{
		name:      name,
		checkFunc: checkFunc,
	}
}

// Check 헬스 체크 실행
func (khc *KeyringHealthCheck) Check(ctx context.Context) error {
	return khc.checkFunc(ctx)
}

// Name 헬스 체크 이름 반환
func (khc *KeyringHealthCheck) Name() string {
	return khc.name
}
