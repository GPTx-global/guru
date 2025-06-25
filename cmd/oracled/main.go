package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/daemon"
	"github.com/GPTx-global/guru/oracle/log"
)

func main() {
	log.InitLogger()
	config.Load()
	config.Print()
	log.ResetLogger(config.Home())

	ctx, cancel := context.WithCancel(context.Background())

	daemon, err := daemon.New(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to create daemon: %w", err))
	}

	if err := daemon.Start(); err != nil {
		panic(fmt.Errorf("failed to start daemon: %w", err))
	}

	fmt.Println()
	fmt.Printf("\t------------------\n")
	fmt.Printf("\t| daemon started |\n")
	fmt.Printf("\t------------------\n")
	fmt.Println()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	cancel()
	daemon.Stop()
	time.Sleep(1 * time.Second)
}
