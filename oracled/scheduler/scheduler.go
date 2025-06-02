package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

type Scheduler struct {
	EventCh  <-chan coretypes.ResultEvent
	ResultCh chan string
	client   *http.Client
	ctx      context.Context
}

func NewScheduler() *Scheduler {
	client := &http.Client{}
	ctx := context.Background()

	return &Scheduler{
		EventCh:  nil,
		ResultCh: make(chan string, 64),
		client:   client,
		ctx:      ctx,
	}
}

func (s *Scheduler) Start() {
	go s.eventProcessor()
}

func (s *Scheduler) eventProcessor() {
	for event := range s.EventCh {
		// fmt.Println("event: ", event)
		_ = event
		// event to job
		job := &Job{
			ID: "temp",
			// URL:  "https://api.coinbase.com/v2/prices/BTC-USD/spot",
			URL: "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT",
			// Path: "data.amount",
			Path: "price",
		}

		req, err := http.NewRequestWithContext(s.ctx, "GET", job.URL, nil)
		if err != nil {
			fmt.Println("failed to create request: ", err)
			continue
		}

		req.Header.Set("User-Agent", "Oracle-Daemon")
		// req.Header.Set("Accept", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			fmt.Println("failed to do request: ", err)
			continue
		}

		// fmt.Println("resp: ", resp)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("failed to read response body: ", err)
			continue
		}

		// fmt.Println("body: ", string(body))

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Println("failed to unmarshal response body: ", err)
			continue
		}

		// price, ok := result["data"].(map[string]interface{})["amount"].(string)
		price, ok := result["price"].(string)
		if !ok {
			fmt.Println("failed to get price from response body: ", err)
			continue
		}

		fmt.Println("price: ", price)

		s.ResultCh <- price
		resp.Body.Close()
	}
}
