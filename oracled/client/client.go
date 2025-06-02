package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	cmtypes "github.com/tendermint/tendermint/types"
)

type Client struct {
	config      *Config
	rpcClient   *http.HTTP
	isConnected bool
	EventCh     chan coretypes.ResultEvent
	ResultCh    <-chan string
}

func NewClient() *Client {
	return &Client{
		config:      LoadConfig(),
		rpcClient:   nil,
		isConnected: false,
		EventCh:     make(chan coretypes.ResultEvent, 64),
		ResultCh:    nil,
	}
}

func (c *Client) Connect() error {
	if c.isConnected {
		return nil
	}

	var err error
	c.rpcClient, err = http.New(c.config.rpcEndpoint, "/websocket")
	if err != nil {
		return errors.New("failed to create rpc client: " + err.Error())
	}
	fmt.Println("rpc client created")

	err = c.rpcClient.Start()
	if err != nil {
		return errors.New("failed to start rpc client: " + err.Error())
	}
	fmt.Println("rpc client started")

	c.isConnected = true

	return nil
}

func (c *Client) Disconnect() error {
	if !c.isConnected {
		return nil
	}

	c.rpcClient.Stop()
	c.rpcClient = nil
	c.isConnected = false

	return nil
}

func (c *Client) Monitor(ctx context.Context) error {
	if !c.isConnected {
		return errors.New("not connected to rpc")
	}

	queryBlock := "tm.event='NewBlock'"
	tempCh, err := c.rpcClient.Subscribe(ctx, "subscribe", queryBlock)
	if err != nil {
		return errors.New("failed to subscribe: " + err.Error())
	}

	for {
		select {
		case event := <-tempCh:
			if event.Data != nil {
				if block, ok := event.Data.(cmtypes.EventDataNewBlock); ok {
					fmt.Printf("새로운 블록 생성됨: 높이=%d, 해시=%X\n",
						block.Block.Height,
						block.Block.Hash())
				}
				c.EventCh <- event
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *Client) PrepareOracle(ctx context.Context) {
	for {
		select {
		case result := <-c.ResultCh:
			fmt.Println("result: ", result)
		case <-ctx.Done():
			return
		}
	}
}
