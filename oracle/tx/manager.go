package tx

import (
	"fmt"
	"sync"

	"github.com/GPTx-global/guru/oracle/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

type TxManager struct {
	clientCtx      client.Context
	resultQueue    chan *types.JobResult
	quit           chan struct{}
	wg             sync.WaitGroup
	accountNumber  uint64
	sequenceNumber uint64
	sequenceLock   sync.Mutex
}

func NewTxManager(clientCtx client.Context) *TxManager {
	txm := new(TxManager)
	txm.clientCtx = clientCtx
	txm.resultQueue = make(chan *types.JobResult, 128)
	txm.quit = make(chan struct{})
	txm.wg = sync.WaitGroup{}
	acc, seq, err := txm.clientCtx.AccountRetriever.GetAccountNumberSequence(txm.clientCtx, txm.clientCtx.GetFromAddress())
	if err != nil {
		panic(err)
	}
	txm.accountNumber = acc
	txm.sequenceNumber = seq
	txm.sequenceLock = sync.Mutex{}

	return txm
}

// TODO: 채널 어떻게 처리할지 고민
func (txm *TxManager) ResultQueue() chan<- *types.JobResult {
	return txm.resultQueue
}

func (txm *TxManager) BuildSubmitTx() ([]byte, error) {
	msgs := make([]sdk.Msg, 0, 1)

	jobResult := <-txm.resultQueue

	dataSet := new(oracletypes.SubmitDataSet)
	dataSet.RequestId = jobResult.ID
	dataSet.RawData = jobResult.Data
	dataSet.Nonce = jobResult.Nonce
	dataSet.Provider = txm.clientCtx.GetFromAddress().String()
	dataSet.Signature = "test"

	msg := new(oracletypes.MsgSubmitOracleData)
	msg.AuthorityAddress = txm.clientCtx.GetFromAddress().String()
	msg.DataSet = dataSet

	msgs = append(msgs, msg)

	gasPrice, err := sdk.ParseDecCoin(types.Config.GasPrice())
	if err != nil {
		return nil, fmt.Errorf("failed to parse gas price: %w", err)
	}

	txm.sequenceLock.Lock()
	factory := tx.Factory{}.
		WithTxConfig(txm.clientCtx.TxConfig).
		WithAccountRetriever(txm.clientCtx.AccountRetriever).
		WithKeybase(txm.clientCtx.Keyring).
		WithChainID(types.Config.ChainID()).
		WithGas(types.Config.GasLimit()).
		WithGasAdjustment(1.2).
		WithGasPrices(gasPrice.String()).
		WithAccountNumber(txm.accountNumber).
		WithSequence(txm.sequenceNumber).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)
	txm.sequenceLock.Unlock()

	txBuilder, err := factory.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to build unsigned tx: %w", err)
	}

	if err := tx.Sign(factory, types.Config.KeyName(), txBuilder, true); err != nil {
		return nil, fmt.Errorf("failed to sign tx: %w", err)
	}

	txBytes, err := txm.clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("failed to encode tx: %w", err)
	}

	return txBytes, nil
}

func (txm *TxManager) BroadcastTx(txBytes []byte) (*sdk.TxResponse, error) {
	res, err := txm.clientCtx.BroadcastTx(txBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast tx: %w", err)
	}

	return res, nil
}
