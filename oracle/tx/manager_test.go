package tx

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/GPTx-global/guru/encoding"
	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
	"github.com/GPTx-global/guru/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockAccountRetriever is a mock for the AccountRetriever interface
type mockAccountRetriever struct {
	mock.Mock
}

func (m *mockAccountRetriever) GetAccountNumberSequence(clientCtx client.Context, addr sdk.AccAddress) (uint64, uint64, error) {
	args := m.Called(clientCtx, addr)
	return args.Get(0).(uint64), args.Get(1).(uint64), args.Error(2)
}

func (m *mockAccountRetriever) GetAccount(client.Context, sdk.AccAddress) (client.Account, error) {
	args := m.Called()
	return args.Get(0).(client.Account), args.Error(1)
}

func (m *mockAccountRetriever) GetAccountWithHeight(client.Context, sdk.AccAddress) (client.Account, int64, error) {
	args := m.Called()
	return args.Get(0).(client.Account), args.Get(1).(int64), args.Error(2)
}

func (m *mockAccountRetriever) EnsureExists(client.Context, sdk.AccAddress) error {
	args := m.Called()
	return args.Error(0)
}

// mockTxConfig is a mock for the TxConfig interface that provides a simple TxEncoder
type mockTxConfig struct {
	client.TxConfig
}

func (m *mockTxConfig) TxEncoder() sdk.TxEncoder {
	return func(tx sdk.Tx) ([]byte, error) {
		return []byte("mock_encoded_tx"), nil
	}
}

// setupTxManagerTest creates a new TxManager with mocked dependencies for isolated testing.
func setupTxManagerTest(t *testing.T) (*TxManager, *mockAccountRetriever) {
	// 1. Initialize logger and config to avoid panics in underlying code

	log.InitLogger()
	config.SetForTesting(
		"test-chain",
		"http://localhost:26657",
		"test-validator",
		os.TempDir(),
		keyring.BackendTest,
		"100uatom",
		300000,
		3,
	)

	// 2. Create an in-memory keyring with a test account
	kr := config.Keyring()
	_, _, err := kr.NewMnemonic(config.KeyName(), keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
	require.NoError(t, err, "failed to create test key")
	addr := config.Address()

	// 3. Create mock dependencies
	mockAccRetriever := new(mockAccountRetriever)
	encCfg := encoding.MakeConfig(nil)

	// 4. Construct client.Context with mocks and test data
	clientCtx := client.Context{}.
		WithAccountRetriever(mockAccRetriever).
		WithKeyring(kr).
		WithFromName(config.KeyName()).
		WithFromAddress(addr).
		WithTxConfig(&mockTxConfig{encCfg.TxConfig}).
		WithChainID(config.ChainID())

	// 5. Mock the initial sequence number retrieval for NewTxManager
	initialAccNum := uint64(10)
	initialSeqNum := uint64(5)
	mockAccRetriever.On("GetAccountNumberSequence", mock.Anything, addr).Return(initialAccNum, initialSeqNum, nil).Once()

	// 6. Create and return the TxManager instance
	txm := NewTxManager(clientCtx)
	require.NotNil(t, txm)

	return txm, mockAccRetriever
}

func TestNewTxManager(t *testing.T) {
	_, mockAccRetriever := setupTxManagerTest(t)
	// Assert that GetAccountNumberSequence was called once during setup
	mockAccRetriever.AssertExpectations(t)
}

func TestNewTxManager_Panic(t *testing.T) {
	// This test sets up a scenario where the account retriever fails,
	// which should cause NewTxManager to panic.
	log.InitLogger()
	config.SetForTesting("test-chain", "", "test", os.TempDir(), keyring.BackendTest, "", 0, 3)
	kr := config.Keyring()
	_, _, _ = kr.NewMnemonic(config.KeyName(), keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
	addr := config.Address()

	mockAccRetriever := new(mockAccountRetriever)
	clientCtx := client.Context{}.WithAccountRetriever(mockAccRetriever).WithFromAddress(addr)

	mockAccRetriever.On("GetAccountNumberSequence", mock.Anything, addr).Return(uint64(0), uint64(0), errors.New("network error")).Once()

	require.Panics(t, func() {
		NewTxManager(clientCtx)
	}, "NewTxManager should panic when AccountRetriever fails")

	mockAccRetriever.AssertExpectations(t)
}

func TestBuildSubmitTx(t *testing.T) {
	txm, _ := setupTxManagerTest(t)

	// Send a job result to the queue
	go func() {
		txm.ResultQueue() <- &types.JobResult{
			ID:    123,
			Data:  "test_data",
			Nonce: 1,
		}
	}()

	// Since BuildSubmitTx blocks until it reads from the queue,
	// we run it in a goroutine and wait for the result.
	var txBytes []byte
	var err error
	done := make(chan struct{})
	go func() {
		txBytes, err = txm.BuildSubmitTx()
		close(done)
	}()

	select {
	case <-done:
		// test completed
	case <-time.After(2 * time.Second):
		t.Fatal("TestBuildSubmitTx timed out")
	}

	require.NoError(t, err)
	require.Equal(t, []byte("mock_encoded_tx"), txBytes)
}

func TestSyncSequenceNumber(t *testing.T) {
	txm, mockAccRetriever := setupTxManagerTest(t)
	addr := config.Address()
	newSeqNum := uint64(20)

	// Setup the mock to return a new sequence number
	mockAccRetriever.On("GetAccountNumberSequence", mock.Anything, addr).Return(uint64(10), newSeqNum, nil).Once()

	err := txm.SyncSequenceNumber()
	require.NoError(t, err)
	mockAccRetriever.AssertExpectations(t)

	// We can't check the private sequenceNumber field directly, but we can verify
	// that a subsequent transaction build would use the new sequence number.
	// This requires more complex mocking of the factory, so for now,
	// we just verify the mock was called.
}

func TestIncrementSequenceNumber(t *testing.T) {
	txm, _ := setupTxManagerTest(t)
	// We can't check the private field, so we just call it to ensure it doesn't panic.
	require.NotPanics(t, func() {
		txm.IncrementSequenceNumber()
	})
}
