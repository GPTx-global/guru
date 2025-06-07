package client

import (
	"context"
	"fmt"
	"sync/atomic"

	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/tendermint/tendermint/rpc/client/http"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/encoding"
	"github.com/GPTx-global/guru/oracled/types"
)

type TxBuilder struct {
	clientCtx client.Context
	config    *Config
	keyring   keyring.Keyring
	sequence  atomic.Uint64
}

func NewTxBuilder(config *Config, rpcClient *http.HTTP) (*TxBuilder, error) {
	encCfg := encoding.MakeConfig(app.ModuleBasics)

	keyRing, err := config.GetKeyring()
	if err != nil {
		return nil, fmt.Errorf("failed to get keyring: %w", err)
	}

	keyInfo, err := keyRing.Key(config.keyName)
	if err != nil {
		return nil, fmt.Errorf("failed to get key %s: %w", config.keyName, err)
	}

	fromAddress, err := keyInfo.GetAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get address from key: %w", err)
	}

	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithTxConfig(encCfg.TxConfig).
		WithLegacyAmino(encCfg.Amino).
		WithKeyring(keyRing).
		WithChainID(config.chainID).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithNodeURI(config.rpcEndpoint).
		WithClient(rpcClient).
		WithFromAddress(fromAddress).
		WithFromName(config.keyName).
		WithBroadcastMode("sync")

	_, seq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get account number sequence: %w", err)
	}

	tb := new(TxBuilder)
	tb.clientCtx = clientCtx
	tb.config = config
	tb.keyring = keyRing
	tb.sequence.Store(seq)

	return tb, nil
}

func (tb *TxBuilder) BuildOracleTx(ctx context.Context, oracleData types.OracleData) ([]byte, error) {
	msgs := make([]sdk.Msg, 0, 1)
	fmt.Printf("oracleData: %+v\n", oracleData)
	msg := &oracletypes.MsgSubmitOracleData{
		FromAddress: tb.clientCtx.GetFromAddress().String(),
		DataSet: &oracletypes.SubmitDataSet{
			RequestId: oracleData.RequestID,
			RawData:   oracleData.Data,
			Nonce:     oracleData.Nonce,
			Provider:  tb.clientCtx.GetFromAddress().String(),
			Signature: "test", // quorum 검증용 시그니처
		},
	}
	// msg := banktypes.NewMsgSend(
	// 	tb.clientCtx.GetFromAddress(),
	// 	tb.clientCtx.GetFromAddress(),
	// 	sdk.NewCoins(sdk.NewCoin("aguru", sdk.NewInt(10))),
	// )
	msgs = append(msgs, msg)

	if len(msgs) == 0 {
		return nil, fmt.Errorf("no messages to send")
	}

	gasPrice, err := sdk.ParseDecCoin(tb.config.gasPrice)
	if err != nil {
		return nil, fmt.Errorf("invalid gas price: %w", err)
	}

	factory := tx.Factory{}.
		WithTxConfig(tb.clientCtx.TxConfig).
		WithAccountRetriever(tb.clientCtx.AccountRetriever).
		WithKeybase(tb.clientCtx.Keyring).
		WithChainID(tb.config.chainID).
		WithGas(tb.config.gasLimit).
		WithGasAdjustment(1.2).
		WithGasPrices(sdk.NewDecCoins(gasPrice).String()).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithSequence(tb.sequence.Load())

	txBuilder, err := factory.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to build tx: %w", err)
	}

	if err := tx.Sign(factory, tb.config.keyName, txBuilder, true); err != nil {
		return nil, fmt.Errorf("failed to sign tx: %w", err)
	}

	txBytes, err := tb.clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("failed to encode tx: %w", err)
	}

	return txBytes, nil
}

func (tb *TxBuilder) BroadcastTx(ctx context.Context, txBytes []byte) (*sdk.TxResponse, error) {
	res, err := tb.clientCtx.BroadcastTx(txBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast tx: %w", err)
	}

	if res.Code != 0 {
		return res, fmt.Errorf("tx failed with code %d: %s", res.Code, res.RawLog)
	}

	return res, nil
}

func (tb *TxBuilder) incSequence() {
	tb.sequence.Add(1)
}
