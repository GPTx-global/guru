package monitor

import (
	"context"
	"fmt"
	"testing"

	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
	"github.com/GPTx-global/guru/oracle/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

// MockQueryClient is a mock implementation of the Oracle query client interface
type MockQueryClient struct {
	mock.Mock
}

// OracleRequestDocs implements the MockQueryClient interface for testing
func (m *MockQueryClient) OracleRequestDocs(ctx context.Context, req *oracletypes.QueryOracleRequestDocsRequest) (*oracletypes.QueryOracleRequestDocsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*oracletypes.QueryOracleRequestDocsResponse), args.Error(1)
}

// OracleRequestDoc implements the MockQueryClient interface for testing
func (m *MockQueryClient) OracleRequestDoc(ctx context.Context, req *oracletypes.QueryOracleRequestDocRequest) (*oracletypes.QueryOracleRequestDocResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*oracletypes.QueryOracleRequestDocResponse), args.Error(1)
}

// MonitorTestSuite defines the test suite for Monitor functionality
type MonitorTestSuite struct {
	suite.Suite
	mockQueryClient *MockQueryClient
	clientContext   client.Context
	backgroundCtx   context.Context
	cancel          context.CancelFunc
}

// SetupSuite runs once before all tests in the suite
func (suite *MonitorTestSuite) SetupSuite() {
	// Initialize logging system
	log.InitLogger()

	// Initialize test configuration
	config.SetForTesting(
		"guru_3110-1",            // chainID
		"http://localhost:26657", // endpoint
		"test-key",               // keyName
		"/tmp/test-keyring",      // keyringDir
		"test",                   // keyringBackend
		"630000000000aguru",      // gasPrices
		30000,                    // gasLimit
	)

	// Initialize background context
	suite.backgroundCtx, suite.cancel = context.WithCancel(context.Background())
}

// TearDownSuite runs once after all tests in the suite complete
func (suite *MonitorTestSuite) TearDownSuite() {
	suite.cancel()
}

// SetupTest runs before each individual test
func (suite *MonitorTestSuite) SetupTest() {
	suite.mockQueryClient = new(MockQueryClient)

	// Set up client context
	suite.clientContext = client.Context{}.
		WithChainID("guru_3110-1").
		WithFromAddress(sdk.AccAddress("test-address"))
}

// TearDownTest runs after each individual test
func (suite *MonitorTestSuite) TearDownTest() {
	suite.mockQueryClient.AssertExpectations(suite.T())
}

// TestNewMonitor tests the New function
func (suite *MonitorTestSuite) TestNewMonitor() {
	monitor := New(suite.clientContext, suite.backgroundCtx)

	suite.NotNil(monitor)
	suite.Equal(suite.clientContext, monitor.clientContext)
	suite.Equal(suite.backgroundCtx, monitor.backgroundContext)
	suite.NotNil(monitor.queryClient)
	suite.Nil(monitor.registerChannel)
	suite.Nil(monitor.updateChannel)
	suite.Nil(monitor.completeChannel)
}

// TestNewMonitorWithQueryClient tests the NewWithQueryClient function
func (suite *MonitorTestSuite) TestNewMonitorWithQueryClient() {
	monitor := NewWithQueryClient(suite.clientContext, suite.backgroundCtx, suite.mockQueryClient)

	suite.NotNil(monitor)
	suite.Equal(suite.clientContext, monitor.clientContext)
	suite.Equal(suite.backgroundCtx, monitor.backgroundContext)
	suite.Equal(suite.mockQueryClient, monitor.queryClient)
	suite.Nil(monitor.registerChannel)
	suite.Nil(monitor.updateChannel)
	suite.Nil(monitor.completeChannel)
}

// TestLoadRequestDocs_Success tests the successful case of LoadRequestDocs
func (suite *MonitorTestSuite) TestLoadRequestDocs_Success() {
	monitor := NewWithQueryClient(suite.clientContext, suite.backgroundCtx, suite.mockQueryClient)

	// Set up expected response in the actual response format
	expectedResponse := &oracletypes.QueryOracleRequestDocsResponse{
		OracleRequestDocs: []*oracletypes.OracleRequestDoc{
			{
				RequestId:   1,
				Status:      oracletypes.RequestStatus_REQUEST_STATUS_ENABLED,
				OracleType:  oracletypes.OracleType_ORACLE_TYPE_CRYPTO,
				Name:        "Test Oracle",
				Description: "Test oracle description",
				Period:      60,
				AccountList: []string{"test-address"},
				Quorum:      1,
				Nonce:       0,
			},
		},
	}

	// Set up mock expectations
	suite.mockQueryClient.On("OracleRequestDocs",
		suite.backgroundCtx,
		&oracletypes.QueryOracleRequestDocsRequest{
			Status: oracletypes.RequestStatus_REQUEST_STATUS_ENABLED,
		}).Return(expectedResponse, nil)

	// Execute test
	docs, err := monitor.LoadRequestDocs()

	// Verify results
	suite.NoError(err)
	suite.NotNil(docs)
	suite.Len(docs, 1)
	suite.Equal(uint64(1), docs[0].RequestId)
	suite.Equal("Test Oracle", docs[0].Name)
}

// TestLoadRequestDocs_Error tests the error case of LoadRequestDocs
func (suite *MonitorTestSuite) TestLoadRequestDocs_Error() {
	monitor := NewWithQueryClient(suite.clientContext, suite.backgroundCtx, suite.mockQueryClient)

	// Set up mock error
	suite.mockQueryClient.On("OracleRequestDocs",
		suite.backgroundCtx,
		&oracletypes.QueryOracleRequestDocsRequest{
			Status: oracletypes.RequestStatus_REQUEST_STATUS_ENABLED,
		}).Return(nil, fmt.Errorf("query failed"))

	// Execute test
	docs, err := monitor.LoadRequestDocs()

	// Verify results
	suite.Error(err)
	suite.Nil(docs)
	suite.Contains(err.Error(), "failed to load oracle request documents")
}

// TestSubscribe_CompleteEvent tests the Subscribe method with complete events
func (suite *MonitorTestSuite) TestSubscribe_CompleteEvent() {
	monitor := NewWithQueryClient(suite.clientContext, suite.backgroundCtx, suite.mockQueryClient)

	// Create channels
	registerCh := make(chan coretypes.ResultEvent)
	updateCh := make(chan coretypes.ResultEvent)
	completeCh := make(chan coretypes.ResultEvent, 1)

	monitor.registerChannel = registerCh
	monitor.updateChannel = updateCh
	monitor.completeChannel = completeCh

	// Create test event
	testEvent := coretypes.ResultEvent{
		Events: map[string][]string{
			types.CompleteID:    {"3"},
			types.CompleteNonce: {"5"},
		},
	}

	// Send event to channel
	completeCh <- testEvent

	// Test Subscribe
	result := monitor.Subscribe()

	// Verify results
	suite.Equal(testEvent, result)
}

// TestMonitorSuite runs the test suite
func TestMonitorSuite(t *testing.T) {
	suite.Run(t, new(MonitorTestSuite))
}
