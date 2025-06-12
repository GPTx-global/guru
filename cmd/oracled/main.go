package main

import (
	"context"
	"fmt"

	"github.com/GPTx-global/guru/oracle/daemon"
	"github.com/GPTx-global/guru/oracle/types"
)

func main() {
	types.LoadConfig()

	ctx := context.Background()
	daemon, err := daemon.New(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to create daemon: %w", err))
	}
	if err := daemon.Start(); err != nil {
		panic(fmt.Errorf("failed to start daemon: %w", err))
	}

	defer daemon.Stop()

	select {}
}
