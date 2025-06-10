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

	client := client.NewClient()
	scheduler := scheduler.NewScheduler()

	scheduler.SetEventChannel(client.GetEventChannel())
	client.SetResultChannel(scheduler.GetResultChannel())

	if err := client.Start(ctx); err != nil {
		fmt.Printf("[ERROR   ] main: Failed to start client: %v\n", err)
		return
	}

	scheduler.Start(ctx)

	fmt.Printf("[SUCCESS] Oracle Daemon started successfully!\n")
	fmt.Printf("=================================================================================================\n")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	fmt.Printf("\n[RESULT ] Oracle Daemon: Received shutdown signal\n")

	cancel()
	fmt.Printf("[  END  ] Oracle Daemon: Graceful shutdown completed\n")
}
