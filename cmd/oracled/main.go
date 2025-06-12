package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GPTx-global/guru/oracle/daemon"
	"github.com/GPTx-global/guru/oracle/types"
)

func main() {
	if err := types.LoadConfig(); err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	daemon, err := daemon.New(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to create daemon: %w", err))
	}

	if err := daemon.Start(); err != nil {
		panic(fmt.Errorf("failed to start daemon: %w", err))
	}

	go daemon.Monitor()
	go daemon.ServeOracle()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	cancel()
	daemon.Stop()
	time.Sleep(3 * time.Second)
}
