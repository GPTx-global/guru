package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GPTx-global/guru/oracled/client"
	"github.com/GPTx-global/guru/oracled/health"
	"github.com/GPTx-global/guru/oracled/scheduler"
)

type OracleDaemon struct {
	client        *client.Client
	scheduler     *scheduler.Scheduler
	healthChecker *health.HealthChecker
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewOracleDaemon() *OracleDaemon {
	ctx, cancel := context.WithCancel(context.Background())

	return &OracleDaemon{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (od *OracleDaemon) Start() error {
	fmt.Printf("[ START ] Oracle Daemon main\n")

	// 간단한 초기화와 시작
	if err := od.initializeAndStart(); err != nil {
		fmt.Printf("[ERROR   ] main: Failed to start daemon: %v\n", err)
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

	// 간단한 헬스 체커 생성 (모니터링용, 자동 재시작 없음)
	od.healthChecker = health.NewHealthChecker(30 * time.Second)
	od.addBasicHealthChecks()

	// 클라이언트와 스케줄러 연결
	od.scheduler.SetJobChannel(od.client.GetJobChannel())
	od.client.SetResultChannel(od.scheduler.GetResultChannel())

	// 클라이언트 시작
	if err := od.client.Start(od.ctx); err != nil {
		return fmt.Errorf("failed to start client: %w", err)
	}

	// 스케줄러 시작
	od.scheduler.Start(od.ctx)

	// 백그라운드 헬스 체크 시작 (로깅만)
	go od.simpleHealthMonitor()

	fmt.Printf("[  END  ] initializeAndStart: SUCCESS\n")
	return nil
}

func (od *OracleDaemon) addBasicHealthChecks() {
	// 기본적인 RPC 연결 헬스 체크만 추가
	rpcHealthCheck := health.NewRPCHealthCheck(
		"rpc_connection",
		od.client.GetHealthCheckFunc(),
	)
	od.healthChecker.AddCheck(rpcHealthCheck)

	// 스케줄러 헬스 체크
	schedulerHealthCheck := health.NewRPCHealthCheck(
		"scheduler",
		od.scheduler.GetHealthCheckFunc(),
	)
	od.healthChecker.AddCheck(schedulerHealthCheck)
}

func (od *OracleDaemon) simpleHealthMonitor() {
	fmt.Printf("[ START ] simpleHealthMonitor\n")

	ticker := time.NewTicker(5 * time.Minute) // 5분마다 상태 로깅
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			od.logBasicStatus()

		case <-od.ctx.Done():
			fmt.Printf("[  END  ] simpleHealthMonitor: context cancelled\n")
			return
		}
	}
}

func (od *OracleDaemon) logBasicStatus() {
	status := od.healthChecker.GetStatus()
	healthyCount := 0
	totalCount := len(status)

	for name, healthStatus := range status {
		if healthStatus.Healthy {
			healthyCount++
		} else {
			fmt.Printf("[  WARN ] Health check: %s is unhealthy - %v\n", name, healthStatus.LastError)
		}
	}

	fmt.Printf("[ HEALTH] System status: %d/%d checks passing\n", healthyCount, totalCount)

	// 스케줄러 상태 로깅
	if od.scheduler != nil {
		activeJobs := od.scheduler.GetActiveJobsCount()
		fmt.Printf("[ STATUS ] Active jobs: %d\n", activeJobs)
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

	// 메인 컨텍스트 취소로 모든 고루틴에 종료 신호 전송
	od.cancel()

	// 컴포넌트들이 정리될 시간 제공
	time.Sleep(3 * time.Second)

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
