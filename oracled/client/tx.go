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
	accNum    uint64
}

func NewTxBuilder(config *Config, rpcClient *http.HTTP) (*TxBuilder, error) {
	fmt.Printf("[ START ] NewTxBuilder - ChainID: %s, KeyName: %s\n",
		config.chainID, config.keyName)

	encCfg := encoding.MakeConfig(app.ModuleBasics)

	keyRing, err := config.GetKeyring()
	if err != nil {
		fmt.Printf("[  END  ] NewTxBuilder: ERROR - failed to get keyring: %v\n", err)
		return nil, fmt.Errorf("failed to get keyring: %w", err)
	}
	// fmt.Printf("[INFO    ] NewTxBuilder: Keyring created successfully\n")

	keyInfo, err := keyRing.Key(config.keyName)
	if err != nil {
		fmt.Printf("[  END  ] NewTxBuilder: ERROR - failed to get key %s: %v\n",
			config.keyName, err)
		return nil, fmt.Errorf("failed to get key %s: %w", config.keyName, err)
	}
	// fmt.Printf("[INFO    ] NewTxBuilder: Key info retrieved for %s\n", config.keyName)

	fromAddress, err := keyInfo.GetAddress()
	if err != nil {
		fmt.Printf("[  END  ] NewTxBuilder: ERROR - failed to get address from key: %v\n", err)
		return nil, fmt.Errorf("failed to get address from key: %w", err)
	}
	// fmt.Printf("[INFO    ] NewTxBuilder: Address retrieved: %s\n", fromAddress.String())

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

	num, seq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, fromAddress)
	if err != nil {
		fmt.Printf("[  END  ] NewTxBuilder: ERROR - failed to get account number sequence: %v\n", err)
		return nil, fmt.Errorf("failed to get account number sequence: %w", err)
	}
	// fmt.Printf("[INFO    ] NewTxBuilder: Account info - Number: %d, Sequence: %d\n", num, seq)

	tb := new(TxBuilder)
	tb.clientCtx = clientCtx
	tb.config = config
	tb.keyring = keyRing
	tb.sequence.Store(seq)
	tb.accNum = num

	fmt.Printf("[  END  ] NewTxBuilder: SUCCESS\n")
	return tb, nil
}

func (tb *TxBuilder) BuildOracleTx(ctx context.Context, oracleData types.OracleData) ([]byte, error) {
	fmt.Printf("[ START ] BuildOracleTx - RequestID: %d, Nonce: %d\n",
		oracleData.RequestID, oracleData.Nonce)

	msgs := make([]sdk.Msg, 0, 1)
	// fmt.Printf("[INFO    ] BuildOracleTx: From Address: %s\n", tb.clientCtx.GetFromAddress())
	// fmt.Printf("[INFO    ] BuildOracleTx: Oracle Data - RequestID: %d, Data: %s, Nonce: %d\n",
	//	oracleData.RequestID, oracleData.Data, oracleData.Nonce)

	msg := &oracletypes.MsgSubmitOracleData{
		AuthorityAddress: tb.clientCtx.GetFromAddress().String(),
		DataSet: &oracletypes.SubmitDataSet{
			RequestId: oracleData.RequestID,
			RawData:   oracleData.Data,
			Nonce:     oracleData.Nonce,
			Provider:  tb.clientCtx.GetFromAddress().String(),
			Signature: "test", // quorum 검증용 시그니처
		},
	}
	msgs = append(msgs, msg)
	// fmt.Printf("[INFO    ] BuildOracleTx: MsgSubmitOracleData created\n")

	if len(msgs) == 0 {
		fmt.Printf("[  END  ] BuildOracleTx: ERROR - no messages to send\n")
		return nil, fmt.Errorf("no messages to send")
	}

	gasPrice, err := sdk.ParseDecCoin(tb.config.gasPrice)
	if err != nil {
		fmt.Printf("[  END  ] BuildOracleTx: ERROR - invalid gas price: %v\n", err)
		return nil, fmt.Errorf("invalid gas price: %w", err)
	}
	// fmt.Printf("[INFO    ] BuildOracleTx: Gas price parsed: %s\n", gasPrice.String())

	factory := tx.Factory{}.
		WithTxConfig(tb.clientCtx.TxConfig).
		WithAccountRetriever(tb.clientCtx.AccountRetriever).
		WithKeybase(tb.clientCtx.Keyring).
		WithChainID(tb.config.chainID).
		WithGas(tb.config.gasLimit).
		WithGasAdjustment(1.2).
		WithGasPrices(sdk.NewDecCoins(gasPrice).String()).
		WithAccountNumber(tb.accNum).
		WithSequence(tb.sequence.Load()).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)

	// fmt.Printf("[INFO    ] BuildOracleTx: Transaction factory configured - AccNum: %d, Sequence: %d\n",
	//	tb.accNum, tb.sequence.Load())

	txBuilder, err := factory.BuildUnsignedTx(msgs...)
	if err != nil {
		fmt.Printf("[  END  ] BuildOracleTx: ERROR - failed to build tx: %v\n", err)
		return nil, fmt.Errorf("failed to build tx: %w", err)
	}
	// fmt.Printf("[INFO    ] BuildOracleTx: Unsigned transaction built\n")

	if err := tx.Sign(factory, tb.config.keyName, txBuilder, true); err != nil {
		fmt.Printf("[  END  ] BuildOracleTx: ERROR - failed to sign tx: %v\n", err)
		return nil, fmt.Errorf("failed to sign tx: %w", err)
	}
	// fmt.Printf("[INFO    ] BuildOracleTx: Transaction signed with key: %s\n", tb.config.keyName)

	txBytes, err := tb.clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		fmt.Printf("[  END  ] BuildOracleTx: ERROR - failed to encode tx: %v\n", err)
		return nil, fmt.Errorf("failed to encode tx: %w", err)
	}

	fmt.Printf("[  END  ] BuildOracleTx: SUCCESS - TX size: %d bytes\n", len(txBytes))
	return txBytes, nil
}

func (tb *TxBuilder) BroadcastTx(ctx context.Context, txBytes []byte) (*sdk.TxResponse, error) {
	fmt.Printf("[ START ] BroadcastTx - Size: %d bytes\n", len(txBytes))

	res, err := tb.clientCtx.BroadcastTx(txBytes)
	if err != nil {
		fmt.Printf("[  END  ] BroadcastTx: ERROR - failed to broadcast: %v\n", err)
		return nil, fmt.Errorf("failed to broadcast tx: %w", err)
	}
	// fmt.Printf("[INFO    ] BroadcastTx: Transaction broadcasted\n")

	if res.Code != 0 {
		fmt.Printf("[  END  ] BroadcastTx: ERROR - tx failed with code %d: %s\n",
			res.Code, res.RawLog)
		return res, fmt.Errorf("tx failed with code %d: %s", res.Code, res.RawLog)
	}

	fmt.Printf("[  END  ] BroadcastTx: SUCCESS - TxHash: %s\n", res.TxHash)
	return res, nil
}

func (tb *TxBuilder) incSequence() {
	tb.sequence.Add(1)
	// fmt.Printf("[INFO    ] incSequence: Sequence incremented %d -> %d\n", oldSeq, newSeq)
}
