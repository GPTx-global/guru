package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/GPTx-global/guru/oracled/client"
	"github.com/GPTx-global/guru/oracled/scheduler"
)

func main() {
	fmt.Println("=== Guru Oracle Daemon 시작 ===")

	// 클라이언트 생성 및 연결
	oracleClient := client.NewClient()
	if err := oracleClient.Connect(); err != nil {
		fmt.Printf("RPC 연결 실패: %v\n", err)
		os.Exit(1)
	}
	defer oracleClient.Disconnect()

	// 스케줄러 생성
	oracleScheduler := scheduler.NewScheduler()

	// 채널 교환 - 데이터 생성자가 채널을 소유하고 소비자가 읽기 채널을 받음
	fmt.Println("Connecting channels between Client and Scheduler...")

	// Client가 생성한 EventCh를 Scheduler가 읽기용으로 받음
	oracleScheduler.SetEventChannel(oracleClient.GetEventChannel())

	// Scheduler가 생성한 OracleDataCh를 Client가 읽기용으로 받음
	oracleClient.SetOracleResultChannel(oracleScheduler.GetOracleDataChannel())

	// 스케줄러 시작
	oracleScheduler.Start()

	// 컨텍스트 생성
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Oracle 처리 고루틴 시작
	go func() {
		oracleClient.PrepareOracle(ctx)
	}()

	// 신호 처리를 위한 채널
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Oracle Daemon이 실행 중입니다...")
	fmt.Println("- Client: EventCh 소유 (이벤트 생성자)")
	fmt.Println("- Scheduler: OracleDataCh 소유 (Oracle 데이터 생성자)")
	fmt.Println("- 채널 연결 완료: Client ←→ Scheduler")
	fmt.Println("종료하려면 Ctrl+C를 누르세요")

	// 모니터링 시작 (메인 고루틴에서 실행)
	go func() {
		if err := oracleClient.Monitor(ctx); err != nil {
			fmt.Printf("모니터링 실패: %v\n", err)
			cancel()
		}
	}()

	// 종료 신호 대기
	<-sigCh
	fmt.Println("\n=== Oracle Daemon 종료 중 ===")
	cancel()
}
