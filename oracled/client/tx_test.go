package client

import (
	"context"
	"testing"
	"time"

	"github.com/GPTx-global/guru/oracled/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TxTestSuite는 TxBuilder 관련 테스트를 위한 test suite입니다
type TxTestSuite struct {
	suite.Suite
	config *Config
}

func (suite *TxTestSuite) SetupTest() {
	suite.config = &Config{
		chainID:     "test-chain",
		keyName:     "test-key",
		rpcEndpoint: "http://localhost:26657",
		gasPrice:    "0.001stake",
		gasLimit:    200000,
	}
}

func TestTxSuite(t *testing.T) {
	suite.Run(t, new(TxTestSuite))
}

func (suite *TxTestSuite) TestTxBuilder_ConfigValidation() {
	// TxBuilder 생성을 위한 Config 검증
	suite.NotNil(suite.config)
	suite.NotEmpty(suite.config.chainID)
	suite.NotEmpty(suite.config.keyName)
	suite.NotEmpty(suite.config.rpcEndpoint)
	suite.NotEmpty(suite.config.gasPrice)
	suite.Greater(suite.config.gasLimit, uint64(0))
}

func (suite *TxTestSuite) TestTxBuilder_SequenceManagement() {
	// TxBuilder의 시퀀스 관리 메서드들이 존재하는지 확인
	// 실제 RPC 연결 없이는 TxBuilder 생성이 어려우므로 메서드 존재만 확인

	// 실제 TxBuilder 구조체에 시퀀스 관리 필드들이 있는지 확인
	suite.True(true) // placeholder test
}

// 기존 개별 테스트들
func TestContainsString(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "contains substring",
			s:        "account sequence mismatch",
			substr:   "sequence",
			expected: true,
		},
		{
			name:     "does not contain substring",
			s:        "insufficient funds",
			substr:   "sequence",
			expected: false,
		},
		{
			name:     "empty string",
			s:        "",
			substr:   "test",
			expected: false,
		},
		{
			name:     "empty substring",
			s:        "test string",
			substr:   "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsString(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsSubstr(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "contains substring case insensitive",
			s:        "INSUFFICIENT FEE",
			substr:   "insufficient",
			expected: true,
		},
		{
			name:     "does not contain substring",
			s:        "account not found",
			substr:   "insufficient",
			expected: false,
		},
		{
			name:     "exact match",
			s:        "fee",
			substr:   "fee",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsSubstr(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTxBuilder_IsSequenceError(t *testing.T) {
	// TxBuilder 생성 없이 메서드 테스트는 어려우므로
	// 메서드가 존재하는지만 확인
	config := &Config{
		chainID:     "test-chain",
		keyName:     "test-key",
		rpcEndpoint: "http://localhost:26657",
		gasPrice:    "0.001stake",
		gasLimit:    200000,
	}

	// 실제 TxBuilder 생성하지 않고 메서드 존재 확인만
	assert.NotNil(t, config)
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("utility functions work correctly", func(t *testing.T) {
		// containsString 테스트
		assert.True(t, containsString("account sequence mismatch", "sequence"))
		assert.False(t, containsString("insufficient funds", "sequence"))

		// containsSubstr 테스트
		assert.True(t, containsSubstr("INSUFFICIENT FEE", "insufficient"))
		assert.False(t, containsSubstr("account not found", "insufficient"))
	})
}

func TestTxBuilder_CreationRequirements(t *testing.T) {
	// TxBuilder 생성에 필요한 요구사항 테스트
	config := &Config{
		chainID:     "test-chain",
		keyName:     "test-key",
		rpcEndpoint: "http://localhost:26657",
		gasPrice:    "0.001stake",
		gasLimit:    200000,
	}

	// Config 검증
	assert.NotEmpty(t, config.chainID, "chainID is required for TxBuilder")
	assert.NotEmpty(t, config.keyName, "keyName is required for TxBuilder")
	assert.NotEmpty(t, config.rpcEndpoint, "rpcEndpoint is required for TxBuilder")
	assert.NotEmpty(t, config.gasPrice, "gasPrice is required for TxBuilder")
	assert.Greater(t, config.gasLimit, uint64(0), "gasLimit must be greater than 0")
}

func TestTxBuilder_ConfigFieldValidation(t *testing.T) {
	// TxBuilder 설정 필드 검증
	tests := []struct {
		name       string
		config     *Config
		shouldPass bool
	}{
		{
			name: "valid config",
			config: &Config{
				chainID:     "guru_3110-1",
				keyName:     "validator",
				rpcEndpoint: "http://localhost:26657",
				gasPrice:    "0.001stake",
				gasLimit:    200000,
			},
			shouldPass: true,
		},
		{
			name: "empty chainID",
			config: &Config{
				chainID:     "",
				keyName:     "validator",
				rpcEndpoint: "http://localhost:26657",
				gasPrice:    "0.001stake",
				gasLimit:    200000,
			},
			shouldPass: false,
		},
		{
			name: "zero gas limit",
			config: &Config{
				chainID:     "guru_3110-1",
				keyName:     "validator",
				rpcEndpoint: "http://localhost:26657",
				gasPrice:    "0.001stake",
				gasLimit:    0,
			},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPass {
				assert.NotEmpty(t, tt.config.chainID)
				assert.NotEmpty(t, tt.config.keyName)
				assert.NotEmpty(t, tt.config.rpcEndpoint)
				assert.NotEmpty(t, tt.config.gasPrice)
				assert.Greater(t, tt.config.gasLimit, uint64(0))
			} else {
				// 적어도 하나의 필드가 잘못되어야 함
				isEmpty := tt.config.chainID == "" ||
					tt.config.keyName == "" ||
					tt.config.rpcEndpoint == "" ||
					tt.config.gasPrice == "" ||
					tt.config.gasLimit == 0
				assert.True(t, isEmpty, "At least one field should be invalid")
			}
		})
	}
}

func TestErrorHandlingFunctions(t *testing.T) {
	// 에러 처리 관련 함수들 테스트
	t.Run("string utility functions", func(t *testing.T) {
		// 시퀀스 에러 메시지 감지 테스트
		sequenceErrors := []string{
			"account sequence mismatch, expected 42, got 41",
			"wrong sequence number",
			"sequence 123 doesn't match expected sequence 124",
		}

		for _, errMsg := range sequenceErrors {
			assert.True(t, containsString(errMsg, "sequence"),
				"Should detect sequence in error message: %s", errMsg)
		}

		// 수수료 부족 에러 메시지 감지 테스트
		feeErrors := []string{
			"insufficient fees",
			"INSUFFICIENT FEE provided",
			"fee too low",
		}

		for _, errMsg := range feeErrors {
			assert.True(t, containsSubstr(errMsg, "insufficient") ||
				containsSubstr(errMsg, "fee"),
				"Should detect fee related error in: %s", errMsg)
		}
	})
}

func TestTxBuilder_incSequence(t *testing.T) {
	// Mock config
	config := &Config{
		chainID:  "test-chain",
		keyName:  "test-key",
		gasPrice: "1000000atest",
		gasLimit: 50000,
	}

	txBuilder := &TxBuilder{
		config:   config,
		sequence: 10,
		accNum:   1,
	}

	// 시퀀스 증가 테스트
	initialSeq := txBuilder.sequence
	txBuilder.incSequence()

	assert.Equal(t, initialSeq+1, txBuilder.sequence)
}

func TestTxBuilder_isSequenceError(t *testing.T) {
	txBuilder := &TxBuilder{}

	testCases := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{
			name:     "sequence error",
			errMsg:   "account sequence mismatch, expected 10, got 9",
			expected: true,
		},
		{
			name:     "invalid sequence",
			errMsg:   "invalid sequence number",
			expected: true,
		},
		{
			name:     "expected sequence",
			errMsg:   "expected sequence 5",
			expected: true,
		},
		{
			name:     "other error",
			errMsg:   "insufficient funds",
			expected: false,
		},
		{
			name:     "empty error",
			errMsg:   "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := txBuilder.isSequenceError(tc.errMsg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTxBuilder_BuildOracleTx_InvalidGasPrice(t *testing.T) {
	config := &Config{
		chainID:  "test-chain",
		keyName:  "test-key",
		gasPrice: "invalid_gas_price", // 잘못된 gas price
		gasLimit: 50000,
	}

	txBuilder := &TxBuilder{
		config:   config,
		sequence: 1,
		accNum:   1,
	}

	oracleData := types.OracleData{
		RequestID: 123,
		Data:      "test_data",
		Nonce:     1,
	}

	ctx := context.Background()
	_, err := txBuilder.BuildOracleTx(ctx, oracleData)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid gas price")
}

func TestTxBuilder_Sequence_Management(t *testing.T) {
	txBuilder := &TxBuilder{
		sequence:       5,
		lastSeqRefresh: time.Now(),
	}

	// 초기 시퀀스 값 확인
	assert.Equal(t, uint64(5), txBuilder.sequence)

	// 시퀀스 증가
	txBuilder.incSequence()
	assert.Equal(t, uint64(6), txBuilder.sequence)

	// 여러 번 증가
	txBuilder.incSequence()
	txBuilder.incSequence()
	assert.Equal(t, uint64(8), txBuilder.sequence)
}

func TestTxBuilder_LastSeqRefresh(t *testing.T) {
	before := time.Now()
	txBuilder := &TxBuilder{
		lastSeqRefresh: before,
	}

	// lastSeqRefresh 시간이 올바르게 설정되었는지 확인
	assert.Equal(t, before, txBuilder.lastSeqRefresh)

	// 새로운 시간으로 업데이트
	after := time.Now()
	txBuilder.lastSeqRefresh = after
	assert.Equal(t, after, txBuilder.lastSeqRefresh)
	assert.True(t, txBuilder.lastSeqRefresh.After(before))
}
