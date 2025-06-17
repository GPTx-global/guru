package tx

import (
	"sync"
	"testing"
	"time"

	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/stretchr/testify/suite"
)

type TxManagerTestSuite struct {
	suite.Suite
}

func TestTxManagerTestSuite(t *testing.T) {
	suite.Run(t, new(TxManagerTestSuite))
}

// Test NewTxManager with proper config loading
func (suite *TxManagerTestSuite) TestNewTxManager() {
	// Load config first to avoid nil pointer
	err := config.LoadConfig()
	suite.Require().NoError(err, "Failed to load config for testing")

	// NewTxManager requires a valid clientCtx with AccountRetriever
	// We test that it doesn't panic when properly initialized
	suite.Run("construction behavior", func() {
		// This will panic due to missing AccountRetriever, which is expected
		suite.Panics(func() {
			_ = NewTxManager(client.Context{})
		})
	})
}

// Test mock TxManager structure directly
func (suite *TxManagerTestSuite) TestTxManagerStructure() {
	// Create mock TxManager to test structure
	txm := &TxManager{
		sequenceNumber: 100,
		accountNumber:  200,
		resultQueue:    make(chan *types.JobResult, 50),
		clientCtx:      client.Context{},
		quit:           make(chan struct{}),
		wg:             sync.WaitGroup{},
		sequenceLock:   sync.Mutex{},
	}

	suite.Equal(uint64(100), txm.sequenceNumber)
	suite.Equal(uint64(200), txm.accountNumber)
	suite.NotNil(txm.resultQueue)
	suite.Equal(50, cap(txm.resultQueue))
	suite.NotNil(txm.quit)
}

// Test ResultQueue method
func (suite *TxManagerTestSuite) TestResultQueue() {
	txm := &TxManager{
		resultQueue: make(chan *types.JobResult, 10),
	}

	queue := txm.ResultQueue()
	suite.NotNil(queue)

	// Test channel operations
	result := &types.JobResult{
		ID:    1,
		Data:  "test",
		Nonce: 1,
	}

	queue <- result

	// Verify result was queued
	suite.Equal(1, len(txm.resultQueue))

	// Read from internal queue
	received := <-txm.resultQueue
	suite.Equal(result, received)
}

// Test IncrementSequenceNumber
func (suite *TxManagerTestSuite) TestIncrementSequenceNumber() {
	txm := &TxManager{
		sequenceNumber: 5,
		sequenceLock:   sync.Mutex{},
	}

	initialSeq := txm.sequenceNumber
	txm.IncrementSequenceNumber()

	suite.Equal(initialSeq+1, txm.sequenceNumber)
}

// Test sequence number concurrency
func (suite *TxManagerTestSuite) TestSequenceNumber_ThreadSafety() {
	txm := &TxManager{
		sequenceNumber: 0,
		sequenceLock:   sync.Mutex{},
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			txm.IncrementSequenceNumber()
		}()
	}

	wg.Wait()

	suite.Equal(uint64(numGoroutines), txm.sequenceNumber)
}

// Test BuildSubmitTx with empty queue (blocking behavior)
func (suite *TxManagerTestSuite) TestBuildSubmitTx_EmptyQueue() {
	// Load config first
	err := config.LoadConfig()
	suite.Require().NoError(err)

	txm := &TxManager{
		resultQueue:  make(chan *types.JobResult),
		clientCtx:    client.Context{},
		sequenceLock: sync.Mutex{},
	}

	// Test blocking behavior
	done := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected panic due to incomplete clientCtx
				done <- true
			}
		}()
		_, _ = txm.BuildSubmitTx()
		done <- true
	}()

	// Should block until data is available
	select {
	case <-done:
		suite.Fail("BuildSubmitTx should block on empty queue")
	case <-time.After(50 * time.Millisecond):
		// Expected behavior - it's blocking
	}

	// Provide data to unblock
	go func() {
		txm.resultQueue <- &types.JobResult{ID: 1, Data: "test", Nonce: 1}
	}()

	// Should complete now (with panic due to clientCtx)
	select {
	case <-done:
		// Expected - completed with panic
	case <-time.After(100 * time.Millisecond):
		suite.Fail("BuildSubmitTx should have unblocked")
	}
}

// Test BuildSubmitTx with data
func (suite *TxManagerTestSuite) TestBuildSubmitTx_WithData() {
	// Load config first
	err := config.LoadConfig()
	suite.Require().NoError(err)

	txm := &TxManager{
		resultQueue:    make(chan *types.JobResult, 1),
		clientCtx:      client.Context{},
		sequenceNumber: 1,
		accountNumber:  1,
		sequenceLock:   sync.Mutex{},
	}

	// Add test data
	testResult := &types.JobResult{
		ID:    1,
		Data:  "100.5",
		Nonce: 1,
	}
	txm.resultQueue <- testResult

	// This will panic due to incomplete clientCtx, but tests data consumption
	suite.Panics(func() {
		_, _ = txm.BuildSubmitTx()
	})

	// Verify data was consumed
	suite.Equal(0, len(txm.resultQueue))
}

// Test concurrent ResultQueue operations
func (suite *TxManagerTestSuite) TestResultQueue_Concurrent() {
	txm := &TxManager{
		resultQueue: make(chan *types.JobResult, 100),
	}

	var wg sync.WaitGroup
	numProducers := 5
	jobsPerProducer := 10

	// Start producers
	for i := 0; i < numProducers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			queue := txm.ResultQueue()
			for j := 0; j < jobsPerProducer; j++ {
				queue <- &types.JobResult{
					ID:    uint64(id*jobsPerProducer + j),
					Data:  "test",
					Nonce: uint64(j),
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all jobs were queued
	expectedJobs := numProducers * jobsPerProducer
	suite.Equal(expectedJobs, len(txm.resultQueue))
}

// Test edge cases
func (suite *TxManagerTestSuite) TestEdgeCases() {
	// Test sequence number overflow
	suite.Run("sequence overflow", func() {
		txm := &TxManager{
			sequenceNumber: ^uint64(0) - 1, // Max - 1
			sequenceLock:   sync.Mutex{},
		}

		txm.IncrementSequenceNumber()
		suite.Equal(^uint64(0), txm.sequenceNumber) // Max

		// Overflow to 0
		txm.IncrementSequenceNumber()
		suite.Equal(uint64(0), txm.sequenceNumber)
	})

	// Test with nil channels
	suite.Run("nil result queue", func() {
		txm := &TxManager{
			resultQueue: nil,
		}

		suite.Panics(func() {
			_ = txm.ResultQueue()
		})
	})
}

// Test SyncSequenceNumber logic
func (suite *TxManagerTestSuite) TestSyncSequenceNumber() {
	txm := &TxManager{
		sequenceNumber: 5,
		clientCtx:      client.Context{}, // Incomplete, will cause panic
		sequenceLock:   sync.Mutex{},
	}

	// Should panic due to incomplete clientCtx
	suite.Panics(func() {
		_ = txm.SyncSequenceNumber()
	})
}

// Test BroadcastTx logic
func (suite *TxManagerTestSuite) TestBroadcastTx() {
	txm := &TxManager{
		clientCtx: client.Context{}, // Incomplete, will cause error
	}

	txBytes := []byte("dummy_tx_bytes")
	_, err := txm.BroadcastTx(txBytes)

	// Should return error due to incomplete clientCtx
	suite.Error(err)
}

// Test multiple ResultQueue calls return same channel
func (suite *TxManagerTestSuite) TestResultQueue_SameInstance() {
	txm := &TxManager{
		resultQueue: make(chan *types.JobResult, 5),
	}

	queue1 := txm.ResultQueue()
	queue2 := txm.ResultQueue()

	suite.Same(queue1, queue2)
}

// Test performance with high load
func (suite *TxManagerTestSuite) TestHighLoadPerformance() {
	txm := &TxManager{
		resultQueue:    make(chan *types.JobResult, 1000),
		sequenceNumber: 0,
		sequenceLock:   sync.Mutex{},
	}

	var wg sync.WaitGroup
	numOperations := 500

	start := time.Now()

	// Mix of operations
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			if index%2 == 0 {
				// Increment sequence
				txm.IncrementSequenceNumber()
			} else {
				// Add to queue
				queue := txm.ResultQueue()
				queue <- &types.JobResult{
					ID:    uint64(index),
					Data:  "test",
					Nonce: uint64(index),
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Should complete quickly
	suite.Less(elapsed, time.Second)

	// Verify operations completed
	expectedIncrements := numOperations / 2
	expectedQueueItems := numOperations - expectedIncrements

	suite.Equal(uint64(expectedIncrements), txm.sequenceNumber)
	suite.Equal(expectedQueueItems, len(txm.resultQueue))
}
