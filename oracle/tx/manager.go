package tx

import (
	"fmt"
	"sync"

	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
	"github.com/GPTx-global/guru/oracle/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

type TxManager struct {
	sequenceNumber uint64
	accountNumber  uint64
	sequenceLock   sync.Mutex
	resultQueue    chan *types.JobResult
	clientCtx      client.Context
	quit           chan struct{}
	wg             sync.WaitGroup
}

// NewTxManager creates a new transaction manager with initialized account information
func NewTxManager(clientCtx client.Context) *TxManager {
	txm := &TxManager{
		clientCtx:    clientCtx,
		resultQueue:  make(chan *types.JobResult, 2<<10),
		quit:         make(chan struct{}),
		wg:           sync.WaitGroup{},
		sequenceLock: sync.Mutex{},
	}
	acc, seq, err := txm.clientCtx.AccountRetriever.GetAccountNumberSequence(txm.clientCtx, config.Address())
	if err != nil {
		panic(err)
	}
	txm.accountNumber = acc
	txm.sequenceNumber = seq

	return txm
}

// ResultQueue returns the channel for receiving job results
func (txm *TxManager) ResultQueue() chan<- *types.JobResult {
	return txm.resultQueue
}

// BuildSubmitTx builds a transaction for submitting oracle data to the blockchain
func (txm *TxManager) BuildSubmitTx() ([]byte, error) {
	log.Debugf("start building submit tx")
	msgs := make([]sdk.Msg, 0, 1)

	jobResult := <-txm.resultQueue

	msg := &oracletypes.MsgSubmitOracleData{
		AuthorityAddress: txm.clientCtx.GetFromAddress().String(),
		DataSet: &oracletypes.SubmitDataSet{
			RequestId: jobResult.ID,
			RawData:   jobResult.Data,
			Nonce:     jobResult.Nonce,
			Provider:  txm.clientCtx.GetFromAddress().String(),
			Signature: "test",
		},
	}

	msgs = append(msgs, msg)

	gasPrice, err := sdk.ParseDecCoin(config.GasPrices())
	if err != nil {
		return nil, fmt.Errorf("failed to parse gas price: %w", err)
	}

	// sequence number 사용 전체 과정을 보호
	txm.sequenceLock.Lock()
	defer txm.sequenceLock.Unlock()

	factory := tx.Factory{}.
		WithTxConfig(txm.clientCtx.TxConfig).
		WithAccountRetriever(txm.clientCtx.AccountRetriever).
		WithKeybase(txm.clientCtx.Keyring).
		WithChainID(config.ChainID()).
		WithGas(config.GasLimit()).
		WithGasAdjustment(1.2).
		WithGasPrices(gasPrice.String()).
		WithAccountNumber(txm.accountNumber).
		WithSequence(txm.sequenceNumber).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)

	txBuilder, err := factory.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to build unsigned tx: %w", err)
	}

	if err := tx.Sign(factory, config.KeyName(), txBuilder, true); err != nil {
		return nil, fmt.Errorf("failed to sign tx: %w", err)
	}

	txBytes, err := txm.clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("failed to encode tx: %w", err)
	}

	log.Debugf("end building submit tx, ID: %+v, Nonce: %+v", jobResult.ID, jobResult.Nonce)

	return txBytes, nil
}

// BroadcastTx broadcasts a transaction to the blockchain network
func (txm *TxManager) BroadcastTx(txBytes []byte) (*sdk.TxResponse, error) {
	res, err := txm.clientCtx.BroadcastTx(txBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast tx: %w", err)
	}

	return res, nil
}

// IncrementSequenceNumber safely increments the sequence number for the next transaction
func (txm *TxManager) IncrementSequenceNumber() {
	txm.sequenceLock.Lock()
	txm.sequenceNumber++
	txm.sequenceLock.Unlock()
}

// SyncSequenceNumber synchronizes the sequence number with the blockchain
func (txm *TxManager) SyncSequenceNumber() error {
	txm.sequenceLock.Lock()
	defer txm.sequenceLock.Unlock()

	_, seq, err := txm.clientCtx.AccountRetriever.GetAccountNumberSequence(txm.clientCtx, config.Address())
	if err != nil {
		return fmt.Errorf("failed to get account sequence: %w", err)
	}

	log.Debugf("sequence updated: %d -> %d", txm.sequenceNumber, seq)
	txm.sequenceNumber = seq
	return nil
}
