package client

import (
	"context"
	"fmt"
	"sync"
	"time"

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
	"github.com/GPTx-global/guru/oracled/retry"
	"github.com/GPTx-global/guru/oracled/types"
)

type TxBuilder struct {
	clientCtx client.Context
	config    *Config
	keyring   keyring.Keyring
	sequence  uint64
	accNum    uint64

	// 에러 처리를 위한 추가 필드
	lastSeqRefresh time.Time
	seqMutex       sync.Mutex // 시퀀스 번호 동기화를 위한 뮤텍스
}

func NewTxBuilder(config *Config, rpcClient *http.HTTP) (*TxBuilder, error) {
	fmt.Printf("[ START ] NewTxBuilder - ChainID: %s, KeyName: %s\n",
		config.chainID, config.keyName)

	encCfg := encoding.MakeConfig(app.ModuleBasics)

	// 키링 생성을 재시도 로직으로 래핑
	var keyRing keyring.Keyring
	err := retry.Do(context.Background(), retry.DefaultRetryConfig(),
		func() error {
			var err error
			keyRing, err = config.GetKeyring()
			return err
		},
		retry.DefaultIsRetryable,
	)
	if err != nil {
		fmt.Printf("[  END  ] NewTxBuilder: ERROR - failed to get keyring: %v\n", err)
		return nil, fmt.Errorf("failed to get keyring after retries: %w", err)
	}

	keyInfo, err := keyRing.Key(config.keyName)
	if err != nil {
		fmt.Printf("[  END  ] NewTxBuilder: ERROR - failed to get key %s: %v\n",
			config.keyName, err)
		return nil, fmt.Errorf("failed to get key %s: %w", config.keyName, err)
	}

	fromAddress, err := keyInfo.GetAddress()
	if err != nil {
		fmt.Printf("[  END  ] NewTxBuilder: ERROR - failed to get address from key: %v\n", err)
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
		WithBroadcastMode("block")

	// 계정 정보 조회를 재시도 로직으로 래핑
	var num, seq uint64
	err = retry.Do(context.Background(), retry.NetworkRetryConfig(),
		func() error {
			var err error
			num, seq, err = clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, fromAddress)
			return err
		},
		retry.DefaultIsRetryable,
	)
	if err != nil {
		fmt.Printf("[  END  ] NewTxBuilder: ERROR - failed to get account number sequence: %v\n", err)
		return nil, fmt.Errorf("failed to get account number sequence after retries: %w", err)
	}

	tb := new(TxBuilder)
	tb.clientCtx = clientCtx
	tb.config = config
	tb.keyring = keyRing
	tb.sequence = seq + 1
	tb.accNum = num
	tb.lastSeqRefresh = time.Now()

	fmt.Printf("[  END  ] NewTxBuilder: SUCCESS\n")
	return tb, nil
}

func (tb *TxBuilder) BuildOracleTx(_ context.Context, oracleData types.OracleData) ([]byte, error) {
	fmt.Printf("[ START ] BuildOracleTx - RequestID: %d, Nonce: %d\n",
		oracleData.RequestID, oracleData.Nonce)

	msgs := make([]sdk.Msg, 0, 1)

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

	if len(msgs) == 0 {
		fmt.Printf("[  END  ] BuildOracleTx: ERROR - no messages to send\n")
		return nil, fmt.Errorf("no messages to send")
	}

	gasPrice, err := sdk.ParseDecCoin(tb.config.gasPrice)
	if err != nil {
		fmt.Printf("[  END  ] BuildOracleTx: ERROR - invalid gas price: %v\n", err)
		return nil, fmt.Errorf("invalid gas price: %w", err)
	}

	// 시퀀스 번호 안전하게 획득 (증가시키지 않음)
	tb.seqMutex.Lock()

	// 시퀀스 번호 갱신 확인 (5분마다 또는 에러 발생 시)
	// if time.Since(tb.lastSeqRefresh) > 5*time.Minute {
	// 	if err := tb.refreshSequenceUnsafe(ctx); err != nil {
	// 		fmt.Printf("[  WARN ] BuildOracleTx: Failed to refresh sequence: %v\n", err)
	// 		// 시퀀스 갱신 실패는 치명적이지 않으므로 계속 진행
	// 	}
	// }

	// 현재 시퀀스 번호만 가져오기 (증가시키지 않음)
	currentSeq := tb.sequence

	tb.seqMutex.Unlock()

	factory := tx.Factory{}.
		WithTxConfig(tb.clientCtx.TxConfig).
		WithAccountRetriever(tb.clientCtx.AccountRetriever).
		WithKeybase(tb.clientCtx.Keyring).
		WithChainID(tb.config.chainID).
		WithGas(tb.config.gasLimit).
		WithGasAdjustment(1.2).
		WithGasPrices(sdk.NewDecCoins(gasPrice).String()).
		WithAccountNumber(tb.accNum).
		WithSequence(currentSeq).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)

	txBuilder, err := factory.BuildUnsignedTx(msgs...)
	if err != nil {
		fmt.Printf("[  END  ] BuildOracleTx: ERROR - failed to build tx: %v\n", err)
		return nil, fmt.Errorf("failed to build tx: %w", err)
	}

	if err := tx.Sign(factory, tb.config.keyName, txBuilder, true); err != nil {
		fmt.Printf("[  END  ] BuildOracleTx: ERROR - failed to sign tx: %v\n", err)
		return nil, fmt.Errorf("failed to sign tx: %w", err)
	}

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

	if res.Code != 0 {
		fmt.Printf("[  END  ] BroadcastTx: ERROR - tx failed with code %d: %s\n",
			res.Code, res.RawLog)

		// 시퀀스 에러인 경우 시퀀스 갱신 시도
		if tb.isSequenceError(res.RawLog) {
			fmt.Printf("[  WARN ] BroadcastTx: Sequence error detected, refreshing sequence\n")
			// if refreshErr := tb.refreshSequence(ctx); refreshErr != nil {
			// 	fmt.Printf("[  WARN ] BroadcastTx: Failed to refresh sequence: %v\n", refreshErr)
			// }
		}

		return res, fmt.Errorf("tx failed with code %d: %s", res.Code, res.RawLog)
	}

	// 트랜잭션이 성공적으로 멤풀에 제출되었을 때만 시퀀스 증가
	// tb.incSequence()

	fmt.Printf("[  END  ] BroadcastTx: SUCCESS - TxHash: %s\n", res.TxHash)
	return res, nil
}

// incSequence 시퀀스 번호를 1 증가 (트랜잭션 성공 시에만 호출)
func (tb *TxBuilder) incSequence() {
	tb.seqMutex.Lock()
	defer tb.seqMutex.Unlock()

	oldSeq := tb.sequence
	tb.sequence++
	fmt.Printf("[ UPDATE] incSequence: %d -> %d\n", oldSeq, tb.sequence)
}

// refreshSequence 시퀀스 번호를 블록체인에서 다시 조회하여 갱신
func (tb *TxBuilder) refreshSequence(ctx context.Context) error {
	tb.seqMutex.Lock()
	defer tb.seqMutex.Unlock()
	return tb.refreshSequenceUnsafe(ctx)
}

func (tb *TxBuilder) refreshSequenceUnsafe(ctx context.Context) error {
	fmt.Printf("[ START ] refreshSequence\n")

	fromAddress := tb.clientCtx.GetFromAddress()

	// 계정 정보 재조회를 재시도 로직으로 래핑
	var num, seq uint64
	err := retry.Do(ctx, retry.NetworkRetryConfig(),
		func() error {
			var err error
			num, seq, err = tb.clientCtx.AccountRetriever.GetAccountNumberSequence(tb.clientCtx, fromAddress)
			return err
		},
		retry.DefaultIsRetryable,
	)

	if err != nil {
		fmt.Printf("[  END  ] refreshSequence: ERROR - %v\n", err)
		return fmt.Errorf("failed to refresh sequence: %w", err)
	}

	oldSeq := tb.sequence
	tb.sequence = seq
	tb.accNum = num
	tb.lastSeqRefresh = time.Now()

	fmt.Printf("[ SUCCESS ] refreshSequence: Updated sequence %d -> %d, accNum: %d\n",
		oldSeq, seq, num)
	fmt.Printf("[  END  ] refreshSequence: SUCCESS\n")
	return nil
}

// isSequenceError 시퀀스 관련 에러인지 확인
func (tb *TxBuilder) isSequenceError(errMsg string) bool {
	sequenceErrors := []string{
		"account sequence mismatch",
		"invalid sequence",
		"sequence",
		"nonce",
	}

	for _, seqErr := range sequenceErrors {
		if containsString(errMsg, seqErr) {
			return true
		}
	}
	return false
}

// GetHealthCheckFunc 헬스 체크 함수 반환
func (tb *TxBuilder) GetHealthCheckFunc() func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if tb.keyring == nil {
			return fmt.Errorf("keyring is nil")
		}

		// 키 접근 가능성 확인
		_, err := tb.keyring.Key(tb.config.keyName)
		if err != nil {
			return fmt.Errorf("failed to access key %s: %w", tb.config.keyName, err)
		}

		// 시퀀스가 너무 오래되었는지 확인
		// if time.Since(tb.lastSeqRefresh) > 10*time.Minute {
		// 	return fmt.Errorf("sequence information is stale (last refresh: %v)", tb.lastSeqRefresh)
		// }

		return nil
	}
}

// containsString 문자열 포함 여부 확인
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstr(s, substr))))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
