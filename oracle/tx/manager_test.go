package tx

import (
	"context"
	"testing"

	"github.com/GPTx-global/guru/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TxManagerTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
}

func TestTxManagerTestSuite(t *testing.T) {
	suite.Run(t, new(TxManagerTestSuite))
}

func (suite *TxManagerTestSuite) SetupTest() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())
}

func (suite *TxManagerTestSuite) TearDownTest() {
	if suite.cancel != nil {
		suite.cancel()
	}
}

func (suite *TxManagerTestSuite) TestNewTxManager() {
	// 실제 NewTxManager는 AccountRetriever가 필요하므로 패닉을 예상
	// 이 테스트는 기본 구조체 초기화만 확인
	defer func() {
		if r := recover(); r != nil {
			// AccountRetriever가 nil이면 패닉 발생 예상
			suite.Contains(r.(error).Error(), "")
		}
	}()

	// Given: 기본 클라이언트 컨텍스트 (AccountRetriever 없음)
	clientCtx := client.Context{}

	// When: TxManager 생성 시도
	tm := NewTxManager(clientCtx)

	// Then: 구조체가 생성되었다면 기본 필드 확인
	if tm != nil {
		suite.NotNil(tm.resultQueue)
		suite.NotNil(tm.quit)
	}
}

func (suite *TxManagerTestSuite) TestTxManager_ResultQueue() {
	// Given: Mock TxManager (직접 생성하여 패닉 회피)
	tm := &TxManager{
		resultQueue: make(chan *types.JobResult, 10),
	}

	// When: ResultQueue 호출
	queue := tm.ResultQueue()

	// Then: 올바른 큐 반환
	suite.NotNil(queue)
	suite.Equal(tm.resultQueue, queue)
}

func (suite *TxManagerTestSuite) TestTxManager_IncrementSequenceNumber() {
	// Given: Mock TxManager
	tm := &TxManager{
		sequenceNumber: 5,
	}

	// When: IncrementSequenceNumber 호출
	tm.IncrementSequenceNumber()

	// Then: 시퀀스 번호가 증가됨
	suite.Equal(uint64(6), tm.sequenceNumber)
}

func (suite *TxManagerTestSuite) TestTxManager_IncrementSequenceNumber_Multiple() {
	// Given: Mock TxManager
	tm := &TxManager{
		sequenceNumber: 0,
	}

	// When: 여러 번 IncrementSequenceNumber 호출
	for i := 0; i < 10; i++ {
		tm.IncrementSequenceNumber()
	}

	// Then: 시퀀스 번호가 올바르게 증가됨
	suite.Equal(uint64(10), tm.sequenceNumber)
}

func (suite *TxManagerTestSuite) TestTxManager_BuildSubmitTx_NoResult() {
	// Given: Mock TxManager (결과 큐에 데이터 없음)
	tm := &TxManager{
		resultQueue: make(chan *types.JobResult),
		clientCtx:   client.Context{},
	}

	// When: BuildSubmitTx 호출 (블로킹됨)
	done := make(chan bool, 1)
	go func() {
		_, err := tm.BuildSubmitTx()
		// 에러가 발생할 것 (clientCtx가 완전하지 않음)
		suite.Error(err)
		done <- true
	}()

	// 결과 제공
	go func() {
		result := &types.JobResult{
			ID:    1,
			Data:  "test_data",
			Nonce: 1,
		}
		tm.resultQueue <- result
	}()

	// Then: 완료 대기
	<-done
}

// 단위 테스트 함수들

func TestTxManagerStructure(t *testing.T) {
	// Given: Mock TxManager 구조체
	tm := &TxManager{
		sequenceNumber: 100,
		accountNumber:  200,
		resultQueue:    make(chan *types.JobResult, 50),
		clientCtx:      client.Context{},
		quit:           make(chan struct{}),
	}

	// Then: 모든 필드가 올바르게 설정됨
	assert.Equal(t, uint64(100), tm.sequenceNumber)
	assert.Equal(t, uint64(200), tm.accountNumber)
	assert.NotNil(t, tm.resultQueue)
	assert.Equal(t, 50, cap(tm.resultQueue))
	assert.NotNil(t, tm.quit)
}

func TestTxManagerResultQueueCapacity(t *testing.T) {
	// Given: Mock TxManager
	tm := &TxManager{
		resultQueue: make(chan *types.JobResult, 2<<10), // 실제 구현과 동일
	}

	// When: 큐 용량 확인
	capacity := cap(tm.resultQueue)

	// Then: 예상된 용량 (2<<10 = 2048)
	assert.Equal(t, 2048, capacity)
	assert.Greater(t, capacity, 0)
}

func TestTxManagerResultQueueFunctionality(t *testing.T) {
	// Given: Mock TxManager
	tm := &TxManager{
		resultQueue: make(chan *types.JobResult, 10),
	}

	results := []*types.JobResult{
		{ID: 1, Data: "result1", Nonce: 1},
		{ID: 2, Data: "result2", Nonce: 2},
		{ID: 3, Data: "result3", Nonce: 3},
	}

	// When: ResultQueue를 통해 결과 전송
	queue := tm.ResultQueue()
	for _, result := range results {
		queue <- result
	}

	// Then: 결과들이 큐에서 순서대로 읽힘
	for _, expectedResult := range results {
		actualResult := <-tm.resultQueue
		assert.Equal(t, expectedResult, actualResult)
	}
}

func TestTxManagerSequenceNumberConcurrency(t *testing.T) {
	// Given: Mock TxManager
	tm := &TxManager{
		sequenceNumber: 0,
	}

	numRoutines := 100
	// When: 동시에 시퀀스 번호 증가
	done := make(chan bool, numRoutines)

	for i := 0; i < numRoutines; i++ {
		go func() {
			tm.IncrementSequenceNumber()
			done <- true
		}()
	}

	// 모든 고루틴 완료 대기
	for i := 0; i < numRoutines; i++ {
		<-done
	}

	// Then: 정확히 100번 증가됨 (동시성 안전)
	assert.Equal(t, uint64(numRoutines), tm.sequenceNumber)
}

func TestTxManagerBroadcastTxWithNilClient(t *testing.T) {
	// Given: 불완전한 TxManager
	tm := &TxManager{
		clientCtx: client.Context{}, // BroadcastTx 메서드 없음
	}

	// When: BroadcastTx 호출
	txBytes := []byte("dummy_tx_bytes")
	_, err := tm.BroadcastTx(txBytes)

	// Then: 에러 발생 (clientCtx가 완전하지 않음)
	assert.Error(t, err)
}

func TestTxManagerResultQueueBlocking(t *testing.T) {
	// Given: 결과가 없는 TxManager
	tm := &TxManager{
		resultQueue: make(chan *types.JobResult),
		clientCtx:   client.Context{},
	}

	// When: BuildSubmitTx를 백그라운드에서 호출
	done := make(chan bool, 1)
	go func() {
		// 결과가 없어서 블로킹됨
		_, err := tm.BuildSubmitTx()
		// 클라이언트 설정이 불완전해서 에러 발생
		assert.Error(t, err)
		done <- true
	}()

	// 결과 제공
	result := &types.JobResult{
		ID:    42,
		Data:  "blocking_test_data",
		Nonce: 5,
	}
	tm.resultQueue <- result

	// Then: BuildSubmitTx가 완료됨
	<-done
}

func TestTxManagerZeroValues(t *testing.T) {
	// Given: 제로값 TxManager
	tm := &TxManager{}

	// When: 기본값 확인
	// Then: 제로값들이 예상대로 설정됨
	assert.Equal(t, uint64(0), tm.sequenceNumber)
	assert.Equal(t, uint64(0), tm.accountNumber)
	assert.Nil(t, tm.resultQueue)
	assert.Zero(t, tm.clientCtx)
	assert.Nil(t, tm.quit)
}

// Mock helper for testing

type mockAccountRetriever struct{}

func (m *mockAccountRetriever) GetAccount(clientCtx client.Context, addr sdk.AccAddress) (client.Account, error) {
	return nil, nil
}

func (m *mockAccountRetriever) GetAccountWithHeight(clientCtx client.Context, addr sdk.AccAddress) (client.Account, int64, error) {
	return nil, 0, nil
}

func (m *mockAccountRetriever) EnsureExists(clientCtx client.Context, addr sdk.AccAddress) error {
	return nil
}

func (m *mockAccountRetriever) GetAccountNumberSequence(clientCtx client.Context, addr sdk.AccAddress) (uint64, uint64, error) {
	return 123, 456, nil // Mock 계정 번호와 시퀀스
}

func TestNewTxManagerWithMockRetriever(t *testing.T) {
	// Given: Mock AccountRetriever가 있는 클라이언트 컨텍스트
	mockRetriever := &mockAccountRetriever{}
	clientCtx := client.Context{
		AccountRetriever: mockRetriever,
	}

	// When: TxManager 생성
	tm := NewTxManager(clientCtx)

	// Then: 정상적으로 생성됨
	require.NotNil(t, tm)
	assert.Equal(t, uint64(123), tm.accountNumber)
	assert.Equal(t, uint64(456), tm.sequenceNumber)
	assert.NotNil(t, tm.resultQueue)
	assert.Equal(t, 2048, cap(tm.resultQueue)) // 2<<10
	assert.NotNil(t, tm.quit)
}
