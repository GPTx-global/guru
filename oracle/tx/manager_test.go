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
	"github.com/stretchr/testify/suite"
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

type TxManagerSuite struct {
	suite.Suite
	clientCtx        client.Context
	mockAccRetriever *mockAccountRetriever
	addr             sdk.AccAddress
}

func TestTxManagerSuite(t *testing.T) {
	suite.Run(t, new(TxManagerSuite))
}

func (s *TxManagerSuite) SetupTest() {
	log.InitLogger()
	config.SetForTesting("test-chain", "http://localhost:26657", "test-validator", os.TempDir(), keyring.BackendTest, "100uatom", 300000, 3)

	kr := config.Keyring()
	_, _, err := kr.NewMnemonic(config.KeyName(), keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
	s.Require().NoError(err, "failed to create test key")
	s.addr = config.Address()

	s.mockAccRetriever = new(mockAccountRetriever)
	encCfg := encoding.MakeConfig(nil)

	s.clientCtx = client.Context{}.
		WithAccountRetriever(s.mockAccRetriever).
		WithKeyring(kr).
		WithFromName(config.KeyName()).
		WithFromAddress(s.addr).
		WithTxConfig(&mockTxConfig{encCfg.TxConfig}).
		WithChainID(config.ChainID())
}

func (s *TxManagerSuite) TestNewTxManager() {
	s.mockAccRetriever.On("GetAccountNumberSequence", mock.Anything, s.addr).Return(uint64(10), uint64(5), nil).Once()
	txm := NewTxManager(s.clientCtx)
	s.Require().NotNil(txm)
	s.mockAccRetriever.AssertExpectations(s.T())
}

func (s *TxManagerSuite) TestNewTxManager_PanicOnFailure() {
	s.mockAccRetriever.On("GetAccountNumberSequence", mock.Anything, s.addr).Return(uint64(0), uint64(0), errors.New("network error")).Once()

	s.Require().Panics(func() {
		NewTxManager(s.clientCtx)
	}, "NewTxManager should panic when AccountRetriever fails")

	s.mockAccRetriever.AssertExpectations(s.T())
}

func (s *TxManagerSuite) TestBuildSubmitTx() {
	s.mockAccRetriever.On("GetAccountNumberSequence", mock.Anything, s.addr).Return(uint64(10), uint64(5), nil).Once()
	txm := NewTxManager(s.clientCtx)

	go func() {
		txm.ResultQueue() <- &types.JobResult{
			ID:    123,
			Data:  "test_data",
			Nonce: 1,
		}
	}()

	var txBytes []byte
	var err error
	done := make(chan struct{})
	go func() {
		txBytes, err = txm.BuildSubmitTx()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		s.T().Fatal("TestBuildSubmitTx timed out")
	}

	s.Require().NoError(err)
	s.Require().Equal([]byte("mock_encoded_tx"), txBytes)
}

func (s *TxManagerSuite) TestSyncSequenceNumber() {
	s.mockAccRetriever.On("GetAccountNumberSequence", mock.Anything, s.addr).Return(uint64(10), uint64(5), nil).Once()
	txm := NewTxManager(s.clientCtx)

	newSeqNum := uint64(20)
	s.mockAccRetriever.On("GetAccountNumberSequence", mock.Anything, s.addr).Return(uint64(10), newSeqNum, nil).Once()

	err := txm.SyncSequenceNumber()
	s.Require().NoError(err)
	s.mockAccRetriever.AssertExpectations(s.T())
}

func (s *TxManagerSuite) TestIncrementSequenceNumber() {
	s.mockAccRetriever.On("GetAccountNumberSequence", mock.Anything, s.addr).Return(uint64(10), uint64(5), nil).Once()
	txm := NewTxManager(s.clientCtx)

	s.Require().NotPanics(func() {
		txm.IncrementSequenceNumber()
	})
}
