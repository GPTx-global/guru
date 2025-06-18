package woker

import (
	"context"
	"testing"
	"time"

	"github.com/GPTx-global/guru/oracle/log"

	"github.com/GPTx-global/guru/oracle/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/stretchr/testify/suite"
)

// mockExecuteJob is a mock implementation of executeJob for testing.
var mockExecuteJob = func(job *types.Job) *types.JobResult {
	if job.Type == types.Complete {
		return &types.JobResult{
			ID:    job.ID,
			Data:  "mock_data",
			Nonce: job.Nonce,
		}
	}
	return nil
}

// originalExecuteJob saves the original executeJob function.
var originalExecuteJob = executeJob

type ManagerTestSuite struct {
	suite.Suite
	jm *JobManager
}

func TestManagerTestSuite(t *testing.T) {
	log.InitLogger()
	suite.Run(t, new(ManagerTestSuite))
}

func (suite *ManagerTestSuite) SetupTest() {
	suite.jm = NewJobManager()
	executeJob = mockExecuteJob // Replace with mock
}

func (suite *ManagerTestSuite) TearDownTest() {
	executeJob = originalExecuteJob // Restore original
}

func (suite *ManagerTestSuite) TestNewJobManager() {
	suite.NotNil(suite.jm)
	suite.NotNil(suite.jm.activeJobs)
	suite.NotNil(suite.jm.jobQueue)
	suite.NotNil(suite.jm.quit)
}

func (suite *ManagerTestSuite) TestSubmitJob() {
	// Fill the queue to capacity
	for i := 0; i < cap(suite.jm.jobQueue); i++ {
		suite.jm.SubmitJob(&types.Job{ID: uint64(i)})
	}

	// Test submitting to a full queue
	// This should be logged as dropped, but the test can't easily check logs.
	// We just confirm it doesn't block or panic.
	suite.jm.SubmitJob(&types.Job{ID: 999})
	suite.Equal(cap(suite.jm.jobQueue), len(suite.jm.jobQueue))
}

func (suite *ManagerTestSuite) TestWorker_JobLifecycle() {
	resultQueue := make(chan *types.JobResult, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	suite.jm.Start(ctx, resultQueue)
	defer suite.jm.Stop()

	// 1. Register a job
	registerJob := &types.Job{
		ID:     1,
		Type:   types.Register,
		Status: oracletypes.RequestStatus_REQUEST_STATUS_ENABLED.String(),
	}
	suite.jm.SubmitJob(registerJob)
	time.Sleep(50 * time.Millisecond) // allow worker to process
	suite.Contains(suite.jm.activeJobs, registerJob.ID)

	// 2. Complete the job
	completeJob := &types.Job{ID: 1, Type: types.Complete}
	suite.jm.SubmitJob(completeJob)

	// Verify result is sent
	select {
	case result := <-resultQueue:
		suite.Equal(uint64(1), result.ID)
		suite.Equal("mock_data", result.Data)
	case <-time.After(1 * time.Second):
		suite.Fail("did not receive job result in time")
	}
}

func (suite *ManagerTestSuite) TestWorker_CompleteDisabledJob() {
	resultQueue := make(chan *types.JobResult, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	suite.jm.Start(ctx, resultQueue)
	defer suite.jm.Stop()

	// Register a disabled job
	disabledJob := &types.Job{
		ID:     2,
		Type:   types.Register,
		Status: oracletypes.RequestStatus_REQUEST_STATUS_DISABLED.String(),
	}
	suite.jm.SubmitJob(disabledJob)
	time.Sleep(50 * time.Millisecond)

	// Attempt to complete it
	suite.jm.SubmitJob(&types.Job{ID: 2, Type: types.Complete})

	// Verify no result is sent
	select {
	case <-resultQueue:
		suite.Fail("should not receive result for disabled job")
	case <-time.After(100 * time.Millisecond):
		// Success
	}
}

func (suite *ManagerTestSuite) TestWorker_CompleteNonExistentJob() {
	resultQueue := make(chan *types.JobResult, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	suite.jm.Start(ctx, resultQueue)
	defer suite.jm.Stop()

	// Attempt to complete a job that was never registered
	suite.jm.SubmitJob(&types.Job{ID: 99, Type: types.Complete})

	// Verify no result is sent
	select {
	case <-resultQueue:
		suite.Fail("should not receive result for non-existent job")
	case <-time.After(100 * time.Millisecond):
		// Success
	}
}
