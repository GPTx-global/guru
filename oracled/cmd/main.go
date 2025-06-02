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
	fmt.Println("----- Guru Oracle Daemon -----\n")

	// 클라이언트와 스케줄러 생성
	oracleClient := client.NewClient()
	oracleScheduler := scheduler.NewScheduler()

	// 채널 교환
	oracleScheduler.SetEventChannel(oracleClient.GetEventChannel())
	oracleClient.SetResultChannel(oracleScheduler.GetResultChannel())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	oracleScheduler.Start(ctx)
	if err := oracleClient.Start(ctx); err != nil {
		fmt.Printf("Client start failed: %v\n", err)
		cancel()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("\n----- Guru Oracle Daemon stopped -----")
	cancel()
}
