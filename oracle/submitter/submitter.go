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

// Submitter handles the creation, signing, and broadcasting of oracle data submission transactions.
// It maintains account state (account number and sequence) and manages transaction lifecycle.
type Submitter struct {
	clientContext  client.Context // Cosmos SDK client context for blockchain interactions
	accountNumber  uint64         // Account number on the blockchain (immutable for account lifetime)
	sequenceNumber uint64         // Current sequence number for transaction ordering
}

// New creates a new Submitter instance and initializes it with the current account state.
// It retrieves the account number and sequence number from the blockchain during initialization.
func New(clientContext client.Context) *Submitter {
	accountNumber, sequenceNumber, err := clientContext.AccountRetriever.GetAccountNumberSequence(clientContext, config.Address())
	if err != nil {
		panic(err)
	}

	return &Submitter{
		clientContext:  clientContext,
		accountNumber:  accountNumber,
		sequenceNumber: sequenceNumber,
	}
}

// BuildTransaction creates an unsigned transaction from a job result.
// It constructs a MsgSubmitOracleData message and builds a transaction with proper gas settings.
// Returns the transaction factory and builder needed for signing.
func (s *Submitter) BuildTransaction(jobResult types.JobResult) (tx.Factory, client.TxBuilder, error) {
	// Create oracle data submission message
	msg := &oracletypes.MsgSubmitOracleData{
		AuthorityAddress: s.clientContext.GetFromAddress().String(),
		DataSet: &oracletypes.SubmitDataSet{
			RequestId: jobResult.ID,
			RawData:   jobResult.Data,
			Nonce:     jobResult.Nonce,
			Provider:  s.clientContext.GetFromAddress().String(),
			Signature: "DATA_SIGNATURE", // TODO: Implement proper data signature
		},
	}

	// Parse gas price from configuration
	gasPrice, err := sdk.ParseDecCoin(config.GasPrices())
	if err != nil {
		return tx.Factory{}, nil, fmt.Errorf("failed to parse gas price: %w", err)
	}

	var sequence uint64
	switch s.clientContext.BroadcastMode {
	case "sync":
		sequence = s.sequenceNumber
	case "block":
		_, sequence, err = s.clientContext.AccountRetriever.GetAccountNumberSequence(s.clientContext, config.Address())
		if err != nil {
			for err != nil {
				_, sequence, err = s.clientContext.AccountRetriever.GetAccountNumberSequence(s.clientContext, config.Address())
			}
		}
	}

	// Create transaction factory with all necessary settings
	factory := tx.Factory{}.
		WithTxConfig(s.clientContext.TxConfig).
		WithAccountRetriever(s.clientContext.AccountRetriever).
		WithKeybase(s.clientContext.Keyring).
		WithChainID(config.ChainID()).
		WithGas(config.GasLimit()).
		WithGasAdjustment(1.3). // 20% gas adjustment for safety
		WithGasPrices(gasPrice.String()).
		WithAccountNumber(s.accountNumber).
		WithSequence(sequence).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)

	// Build unsigned transaction
	txBuilder, err := factory.BuildUnsignedTx(msg)
	if err != nil {
		return tx.Factory{}, nil, fmt.Errorf("failed to build unsigned transaction: %w", err)
	}

	return factory, txBuilder, nil
}

// SignTransaction signs a transaction using the configured private key.
// It takes the factory and builder from BuildTransaction and returns the signed transaction bytes.
func (s *Submitter) SignTransaction(factory tx.Factory, txBuilder client.TxBuilder) ([]byte, error) {
	// Sign the transaction with the configured key
	if err := tx.Sign(factory, config.KeyName(), txBuilder, true); err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Encode the signed transaction to bytes for broadcasting
	txBytes, err := s.clientContext.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("failed to encode transaction: %w", err)
	}

	return txBytes, nil
}

// BroadcastTransaction broadcasts a signed transaction to the blockchain network.
// It handles the response and manages sequence number updates for successful transactions.
// For failed transactions, it attempts to resynchronize the sequence number.
func (s *Submitter) BroadcastTransaction(txBytes []byte) error {
	// Broadcast the transaction to the network
	res, err := s.clientContext.BroadcastTx(txBytes)
	if err != nil {
		return fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Successful transaction
	if res.Code == 0 {
		log.Debugf("transaction broadcasted successfully: %s", res.TxHash)
		s.sequenceNumber++ // Increment sequence for next transaction

		return nil
	}

	// Failed transaction
	failedSequence := s.sequenceNumber
	_, s.sequenceNumber, err = s.clientContext.AccountRetriever.GetAccountNumberSequence(s.clientContext, config.Address())
	if err != nil {
		return fmt.Errorf("failed to get account number sequence: %w", err)
	}

	switch res.Code {
	case 18:
		log.Debugf("already certified")
	case 32:
		log.Debugf("sequence number synchronized: %d -> %d", failedSequence, s.sequenceNumber)
	}

	return fmt.Errorf("code: %d, raw log: %s, tx hash: %s", res.Code, res.RawLog, res.TxHash)
}
