package submitter

import (
	"fmt"

	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
	"github.com/GPTx-global/guru/oracle/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
)

type Submitter struct {
	clientContext  client.Context
	accountNumber  uint64
	sequenceNumber uint64
}

func New(clientContext client.Context) *Submitter {
	acc, seq, err := clientContext.AccountRetriever.GetAccountNumberSequence(clientContext, config.Address())
	if err != nil {
		panic(err)
	}

	return &Submitter{
		clientContext:  clientContext,
		accountNumber:  acc,
		sequenceNumber: seq,
	}
}

func (s *Submitter) BuildTransaction(jobResult types.JobResult) (tx.Factory, client.TxBuilder, error) {
	msg := &oracletypes.MsgSubmitOracleData{
		AuthorityAddress: s.clientContext.GetFromAddress().String(),
		DataSet: &oracletypes.SubmitDataSet{
			RequestId: jobResult.ID,
			RawData:   jobResult.Data,
			Nonce:     jobResult.Nonce,
			Provider:  s.clientContext.GetFromAddress().String(),
			Signature: "DATA_SIGNATURE",
		},
	}

	gasPrice, err := sdk.ParseDecCoin(config.GasPrices())
	if err != nil {
		return tx.Factory{}, nil, fmt.Errorf("failed to parse gas price: %w", err)
	}

	factory := tx.Factory{}.
		WithTxConfig(s.clientContext.TxConfig).
		WithAccountRetriever(s.clientContext.AccountRetriever).
		WithKeybase(s.clientContext.Keyring).
		WithChainID(config.ChainID()).
		WithGas(config.GasLimit()).
		WithGasAdjustment(1.2).
		WithGasPrices(gasPrice.String()).
		WithAccountNumber(s.accountNumber).
		WithSequence(s.sequenceNumber).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)

	txBuilder, err := factory.BuildUnsignedTx(msg)
	if err != nil {
		return tx.Factory{}, nil, fmt.Errorf("failed to build and sign transaction: %w", err)
	}

	return factory, txBuilder, nil
}

func (s *Submitter) SignTransaction(factory tx.Factory, txBuilder client.TxBuilder) ([]byte, error) {
	if err := tx.Sign(factory, config.KeyName(), txBuilder, true); err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	txBytes, err := s.clientContext.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("failed to encode tx: %w", err)
	}

	return txBytes, nil
}

func (s *Submitter) BroadcastTransaction(txBytes []byte) error {
	res, err := s.clientContext.BroadcastTx(txBytes)
	if err != nil {
		return fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	if res.Code == 0 {
		log.Debugf("tx broadcasted: %s", res.TxHash)
		s.sequenceNumber++
	} else {
		_, seq, err := s.clientContext.AccountRetriever.GetAccountNumberSequence(s.clientContext, config.Address())
		if err != nil {
			return fmt.Errorf("failed to get account number sequence: %w", err)
		}

		log.Debugf("sequence number synchronized: %d -> %d", s.sequenceNumber, seq)
		s.sequenceNumber = seq

		return fmt.Errorf("failed to broadcast transaction: %d: %s", res.Code, res.RawLog)
	}

	return nil
}
