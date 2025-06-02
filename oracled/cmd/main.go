package main

import (
	"context"
	"fmt"

	"github.com/GPTx-global/guru/oracled/client"
	"github.com/GPTx-global/guru/oracled/scheduler"
)

func main() {
	client := client.NewClient()
	if err := client.Connect(); err != nil {
		fmt.Println("failed to connect to rpc: ", err)
	}

	scheduler := scheduler.NewScheduler()

	scheduler.EventCh = client.EventCh
	client.ResultCh = scheduler.ResultCh
	scheduler.Start()

	ctx := context.Background()

	go client.PrepareOracle(ctx)
	if err := client.Monitor(ctx); err != nil {
		fmt.Println("failed to monitor: ", err)
	}

	fmt.Println("done")
}
