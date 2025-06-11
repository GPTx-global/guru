package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/GPTx-global/guru/oracled/client"
	"github.com/GPTx-global/guru/oracled/health"
	"github.com/GPTx-global/guru/oracled/retry"
	"github.com/GPTx-global/guru/oracled/scheduler"
)

type OracleDaemon struct {
	client        *client.Client
	scheduler     *scheduler.Scheduler
	healthChecker *health.HealthChecker
	ctx           context.Context
	cancel        context.CancelFunc
	restartMutex  sync.Mutex
	restartCount  int
	maxRestarts   int
}

func NewOracleDaemon() *OracleDaemon {
	ctx, cancel := context.WithCancel(context.Background())

	return &OracleDaemon{
		ctx:         ctx,
		cancel:      cancel,
		maxRestarts: 10, // 최대 10번 재시작 허용
	}
}

func (od *OracleDaemon) Start() error {
	fmt.Printf("[ START ] Oracle Daemon main\n")

	// 전체 시스템 시작을 재시도 로직으로 래핑
	err := retry.Do(od.ctx, retry.DefaultRetryConfig(),
		func() error {
			return od.initializeAndStart()
		},
		retry.DefaultIsRetryable,
	)

	if err != nil {
		fmt.Printf("[ERROR   ] main: Failed to start daemon after retries: %v\n", err)
		return err
	}

	fmt.Printf("[SUCCESS] Oracle Daemon started successfully!\n")
	fmt.Printf("=================================================================================================\n")

	// 시그널 대기 및 우아한 종료
	return od.waitForShutdown()
}

func (od *OracleDaemon) initializeAndStart() error {
	fmt.Printf("[ START ] initializeAndStart\n")

	// Oracle client 생성
	od.client = client.NewClient()
	if od.client == nil {
		return fmt.Errorf("failed to create Oracle client")
	}

	// Scheduler 생성
	od.scheduler = scheduler.NewScheduler()
	if od.scheduler == nil {
		return fmt.Errorf("failed to create Scheduler")
	}

	// 헬스 체커 생성
	od.healthChecker = health.NewHealthChecker(30 * time.Second)

	// 헬스 체크 추가
	od.addHealthChecks()

	// 클라이언트와 스케줄러 연결
	od.scheduler.SetJobChannel(od.client.GetJobChannel())
	od.client.SetResultChannel(od.scheduler.GetResultChannel())

	// 클라이언트 시작
	if err := od.client.Start(od.ctx); err != nil {
		return fmt.Errorf("failed to start client: %w", err)
	}

	// 스케줄러 시작
	od.scheduler.Start(od.ctx)

	// 헬스 체커 시작
	go od.healthChecker.Start(od.ctx)

	// 시스템 모니터링 시작
	go od.systemMonitor()

	fmt.Printf("[  END  ] initializeAndStart: SUCCESS\n")
	return nil
}

func (od *OracleDaemon) addHealthChecks() {
	// RPC 연결 헬스 체크
	rpcHealthCheck := health.NewRPCHealthCheck(
		"rpc_connection",
		od.client.GetHealthCheckFunc(),
	)
	od.healthChecker.AddCheck(rpcHealthCheck)

	// 키링 헬스 체크 (TxBuilder를 통해)
	if od.client != nil {
		// TxBuilder 헬스 체크는 클라이언트 내부에서 관리
	}

	// 스케줄러 헬스 체크
	schedulerHealthCheck := health.NewRPCHealthCheck(
		"scheduler",
		od.scheduler.GetHealthCheckFunc(),
	)
	od.healthChecker.AddCheck(schedulerHealthCheck)
}

func (od *OracleDaemon) systemMonitor() {
	fmt.Printf("[ START ] systemMonitor\n")

	ticker := time.NewTicker(1 * time.Minute) // 1분마다 시스템 상태 확인
	defer ticker.Stop()

	consecutiveFailures := 0
	maxConsecutiveFailures := 5

	for {
		select {
		case <-ticker.C:
			isHealthy := od.healthChecker.IsHealthy()

			if !isHealthy {
				consecutiveFailures++
				fmt.Printf("[  WARN ] systemMonitor: System unhealthy (failures: %d/%d)\n",
					consecutiveFailures, maxConsecutiveFailures)

				// 연속된 실패가 임계값을 초과하면 시스템 재시작 시도
				if consecutiveFailures >= maxConsecutiveFailures {
					fmt.Printf("[  WARN ] systemMonitor: Attempting system restart due to consecutive failures\n")
					go od.attemptRestart()
					consecutiveFailures = 0 // 재시작 시도 후 카운터 리셋
				}
			} else {
				if consecutiveFailures > 0 {
					fmt.Printf("[ HEALTH] systemMonitor: System recovered\n")
				}
				consecutiveFailures = 0
			}

			// 상태 로깅
			od.logSystemStatus()

		case <-od.ctx.Done():
			fmt.Printf("[  END  ] systemMonitor: context cancelled\n")
			return
		}
	}
}

func (od *OracleDaemon) logSystemStatus() {
	status := od.healthChecker.GetStatus()
	healthyCount := 0
	totalCount := len(status)

	for name, healthStatus := range status {
		if healthStatus.Healthy {
			healthyCount++
		} else {
			fmt.Printf("[  WARN ] Health check failed - %s: %v\n", name, healthStatus.LastError)
		}
	}

	fmt.Printf("[ HEALTH] System status: %d/%d checks passing\n", healthyCount, totalCount)

	// 스케줄러 상태 추가 로깅
	if od.scheduler != nil {
		activeJobs := od.scheduler.GetActiveJobsCount()
		fmt.Printf("[ STATUS ] Active jobs: %d\n", activeJobs)
	}
}

func (od *OracleDaemon) attemptRestart() {
	od.restartMutex.Lock()
	defer od.restartMutex.Unlock()

	od.restartCount++

	fmt.Printf("[  WARN ] attemptRestart: Attempt #%d (max: %d)\n", od.restartCount, od.maxRestarts)

	if od.restartCount > od.maxRestarts {
		fmt.Printf("[ERROR   ] attemptRestart: Maximum restart attempts exceeded, shutting down\n")
		od.cancel() // 완전 종료
		return
	}

	// 점진적 재시작 시도
	fmt.Printf("[ RESTART] attemptRestart: Starting gradual restart\n")

	// 1. 클라이언트 재시작 시도
	if od.client != nil && !od.client.IsConnected() {
		fmt.Printf("[ RESTART] attemptRestart: Restarting client\n")

		err := retry.Do(od.ctx, retry.NetworkRetryConfig(),
			func() error {
				return od.client.Start(od.ctx)
			},
			retry.DefaultIsRetryable,
		)

		if err != nil {
			fmt.Printf("[  WARN ] attemptRestart: Failed to restart client: %v\n", err)
		} else {
			fmt.Printf("[ RESTART] attemptRestart: Client restarted successfully\n")
		}
	}

	// 2. 전체 시스템이 여전히 불안정하면 완전 재초기화
	time.Sleep(10 * time.Second) // 재시작 후 안정화 대기

	if !od.healthChecker.IsHealthy() {
		fmt.Printf("[ RESTART] attemptRestart: Performing full system restart\n")

		// 기존 컨텍스트 취소
		od.cancel()

		// 새 컨텍스트 생성
		od.ctx, od.cancel = context.WithCancel(context.Background())

		// 시스템 재초기화
		err := od.initializeAndStart()
		if err != nil {
			fmt.Printf("[ERROR   ] attemptRestart: Full restart failed: %v\n", err)
		} else {
			fmt.Printf("[ RESTART] attemptRestart: Full restart completed successfully\n")
		}
	}
}

func (od *OracleDaemon) waitForShutdown() error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	fmt.Printf("\n[ RESULT] Oracle Daemon: Received shutdown signal\n")

	return od.gracefulShutdown()
}

func (od *OracleDaemon) gracefulShutdown() error {
	fmt.Printf("[ START ] gracefulShutdown\n")

	// 타임아웃을 가진 컨텍스트로 우아한 종료
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 메인 컨텍스트 취소로 모든 고루틴에 종료 신호 전송
	od.cancel()

	// 컴포넌트들이 정리될 시간 제공
	select {
	case <-shutdownCtx.Done():
		fmt.Printf("[  WARN ] gracefulShutdown: Timeout reached, forcing shutdown\n")
	case <-time.After(5 * time.Second):
		fmt.Printf("[ HEALTH] gracefulShutdown: Components shut down gracefully\n")
	}

	fmt.Printf("[  END  ] Oracle Daemon: Graceful shutdown completed\n")
	return nil
}

func main() {
	daemon := NewOracleDaemon()

	if err := daemon.Start(); err != nil {
		fmt.Printf("[ERROR   ] main: Oracle daemon failed to start: %v\n", err)
		os.Exit(1)
	}
}
