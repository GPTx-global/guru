package monitor

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
	"github.com/GPTx-global/guru/oracle/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

// QueryClient defines the interface for oracle query operations.
// It provides methods to query oracle request documents from the blockchain.
type QueryClient interface {
	// OracleRequestDocs queries multiple oracle request documents with filtering options
	OracleRequestDocs(ctx context.Context, req *oracletypes.QueryOracleRequestDocsRequest) (*oracletypes.QueryOracleRequestDocsResponse, error)
	// OracleRequestDoc queries a single oracle request document by its ID
	OracleRequestDoc(ctx context.Context, req *oracletypes.QueryOracleRequestDocRequest) (*oracletypes.QueryOracleRequestDocResponse, error)
}

// queryClientWrapper wraps the gRPC query client to implement our QueryClient interface.
// This allows for easier testing and dependency injection.
type queryClientWrapper struct {
	client oracletypes.QueryClient
}

// OracleRequestDocs implements QueryClient interface by delegating to the wrapped gRPC client.
func (w *queryClientWrapper) OracleRequestDocs(ctx context.Context, req *oracletypes.QueryOracleRequestDocsRequest) (*oracletypes.QueryOracleRequestDocsResponse, error) {
	return w.client.OracleRequestDocs(ctx, req)
}

// OracleRequestDoc implements QueryClient interface by delegating to the wrapped gRPC client.
func (w *queryClientWrapper) OracleRequestDoc(ctx context.Context, req *oracletypes.QueryOracleRequestDocRequest) (*oracletypes.QueryOracleRequestDocResponse, error) {
	return w.client.OracleRequestDoc(ctx, req)
}

// Monitor manages event subscription and monitoring for oracle operations.
// It listens to blockchain events related to oracle request registration, updates, and completion.
type Monitor struct {
	clientContext     client.Context               // Cosmos SDK client context for blockchain interactions
	backgroundContext context.Context              // Background context for long-running operations
	queryClient       QueryClient                  // Client for querying oracle data from blockchain
	registerChannel   <-chan coretypes.ResultEvent // Channel for receiving oracle registration events
	updateChannel     <-chan coretypes.ResultEvent // Channel for receiving oracle update events
	completeChannel   <-chan coretypes.ResultEvent // Channel for receiving oracle completion events
}

// New creates a new Monitor instance with the provided client and background contexts.
// It initializes the monitor with a default query client wrapper.
func New(clientContext client.Context, backgroundContext context.Context) *Monitor {
	return &Monitor{
		clientContext:     clientContext,
		backgroundContext: backgroundContext,
		queryClient:       &queryClientWrapper{client: oracletypes.NewQueryClient(clientContext)},
		registerChannel:   nil,
		updateChannel:     nil,
		completeChannel:   nil,
	}
}

// NewWithQueryClient creates a new Monitor with a custom query client.
// This constructor is primarily used for testing with mock query clients.
func NewWithQueryClient(clientContext client.Context, backgroundContext context.Context, queryClient QueryClient) *Monitor {
	return &Monitor{
		clientContext:     clientContext,
		backgroundContext: backgroundContext,
		queryClient:       queryClient,
		registerChannel:   nil,
		updateChannel:     nil,
		completeChannel:   nil,
	}
}

// Start begins the monitoring process by initializing event subscription channels.
// This method should be called before attempting to subscribe to events.
func (m *Monitor) Start() {
	log.Debugf("monitor manager starting")
	m.initChannels()
}

// Stop stops the monitoring process by closing all event subscription channels.
// After calling this method, the monitor will no longer receive events.
func (m *Monitor) Stop() {
	m.registerChannel = nil
	m.updateChannel = nil
	m.completeChannel = nil
	log.Debugf("monitor manager stopped")
}

// initChannels initializes subscription channels for different types of oracle events.
// It subscribes to register, update, and complete events from the blockchain.
func (m *Monitor) initChannels() {
	var err error

	// Subscribe to oracle request registration events
	registerQuery := "tm.event='Tx' AND message.action='/guru.oracle.v1.MsgRegisterOracleRequestDoc'"
	m.registerChannel, err = m.clientContext.Client.Subscribe(m.backgroundContext, "register", registerQuery, config.ChannelSize())
	if err != nil {
		panic(fmt.Errorf("failed to subscribe to register: %w", err))
	}

	// Subscribe to oracle request update events
	updateQuery := "tm.event='Tx' AND message.action='/guru.oracle.v1.MsgUpdateOracleRequestDoc'"
	m.updateChannel, err = m.clientContext.Client.Subscribe(m.backgroundContext, "update", updateQuery, config.ChannelSize())
	if err != nil {
		panic(fmt.Errorf("failed to subscribe to update: %w", err))
	}

	// Subscribe to oracle data completion events
	completeQuery := fmt.Sprintf("tm.event='NewBlock' AND %s.%s EXISTS", oracletypes.EventTypeCompleteOracleDataSet, oracletypes.AttributeKeyRequestId)
	m.completeChannel, err = m.clientContext.Client.Subscribe(m.backgroundContext, "complete", completeQuery, config.ChannelSize())
	if err != nil {
		panic(fmt.Errorf("failed to subscribe to complete: %w", err))
	}

	log.Debugf("init channels successfully")
}

// LoadRequestDocs retrieves all enabled oracle request documents from the blockchain.
// This method is typically called during startup to load existing oracle requests.
func (m *Monitor) LoadRequestDocs() ([]*oracletypes.OracleRequestDoc, error) {
	res, err := m.queryClient.OracleRequestDocs(m.backgroundContext, &oracletypes.QueryOracleRequestDocsRequest{Status: oracletypes.RequestStatus_REQUEST_STATUS_ENABLED})
	if err != nil {
		log.Errorf("failed to load oracle request documents: %v", err)
		return nil, fmt.Errorf("failed to load oracle request documents: %w", err)
	}

	return res.OracleRequestDocs, nil
}

// Subscribe listens for oracle events and returns the appropriate response.
// It handles three types of events: register, update, and complete.
// Returns nil if the event is not relevant to this oracle instance.
func (m *Monitor) Subscribe() any {
	select {
	case event := <-m.registerChannel:
		// Handle oracle request registration events
		id, err := strconv.ParseUint(event.Events[types.RegisterID][0], 10, 64)
		if err != nil {
			log.Debugf("failed to parse request ID: %v", err)
			return nil
		}

		// Check if this oracle instance is responsible for this request
		accounts := event.Events[types.RegisterAccountList][0]
		if !strings.Contains(accounts, config.Address().String()) {
			log.Debugf("register event not for this oracle instance: %d", id)
			return nil
		}

		// Query the full request document
		queryRes, err := m.queryClient.OracleRequestDoc(m.backgroundContext, &oracletypes.QueryOracleRequestDocRequest{RequestId: id})
		if err != nil {
			log.Debugf("failed to query oracle request document: %v", err)
			return nil
		}

		return queryRes.RequestDoc

	case event := <-m.updateChannel:
		// Handle oracle request update events
		id, err := strconv.ParseUint(event.Events[types.UpdateID][0], 10, 64)
		if err != nil {
			log.Debugf("failed to parse request ID: %v", err)
			return nil
		}

		// Query the updated request document
		queryRes, err := m.queryClient.OracleRequestDoc(m.backgroundContext, &oracletypes.QueryOracleRequestDocRequest{RequestId: id})
		if err != nil {
			log.Debugf("failed to query oracle request document: %v", err)
			return nil
		}

		return queryRes.RequestDoc

	case event := <-m.completeChannel:
		// Handle oracle data completion events
		return event
	}
}
