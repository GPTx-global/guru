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
	fmt.Printf("[ START ] Oracle Daemon main\n")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Oracle client 생성
	// fmt.Printf("[INFO    ] main: Creating Oracle Client\n")
	client := client.NewClient()

	// Scheduler 생성
	// fmt.Printf("[INFO    ] main: Creating Scheduler\n")
	scheduler := scheduler.NewScheduler()

	// 클라이언트와 스케줄러 연결
	// fmt.Printf("[INFO    ] main: Connecting client and scheduler channels\n")
	scheduler.SetEventChannel(client.GetEventChannel())
	client.SetResultChannel(scheduler.GetResultChannel())

	// 클라이언트 시작
	// fmt.Printf("[INFO    ] main: Starting Oracle Client\n")
	if err := client.Start(ctx); err != nil {
		fmt.Printf("[ERROR   ] main: Failed to start client: %v\n", err)
		return
	}

	// 스케줄러 시작
	// fmt.Printf("[INFO    ] main: Starting Scheduler\n")
	scheduler.Start(ctx)

	fmt.Printf("[SUCCESS] Oracle Daemon started successfully!\n")
	fmt.Printf("=================================================================================================\n")

	// 시그널 대기
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	fmt.Printf("\n[RESULT ] Oracle Daemon: Received shutdown signal\n")

	cancel()
	fmt.Printf("[  END  ] Oracle Daemon: Graceful shutdown completed\n")
}
