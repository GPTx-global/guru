package woker

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/GPTx-global/guru/oracle/types"
	"github.com/stretchr/testify/suite"
)

type JobManagerTestSuite struct {
	suite.Suite
	ctx    context.Context
	cancel context.CancelFunc
}

func (suite *JobManagerTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())
}

func (suite *JobManagerTestSuite) TearDownSuite() {
	suite.cancel()
}

func TestJobManagerTestSuite(t *testing.T) {
	suite.Run(t, new(JobManagerTestSuite))
}

// Test NewJobManager
func (suite *JobManagerTestSuite) TestNewJobManager() {
	jm := NewJobManager()

	suite.NotNil(jm)
	suite.NotNil(jm.activeJobs)
	suite.NotNil(jm.jobQueue)
	suite.NotNil(jm.quit)
	suite.Equal(0, len(jm.activeJobs))
	suite.Equal(2<<5, cap(jm.jobQueue))
}

// Test Start method
func (suite *JobManagerTestSuite) TestStart() {
	jm := NewJobManager()
	resultQueue := make(chan *types.JobResult, 10)

	// Start should launch goroutines equal to CPU count
	jm.Start(suite.ctx, resultQueue)

	// Give some time for goroutines to start
	time.Sleep(10 * time.Millisecond)

	// Stop the manager
	jm.Stop()

	// Test should not panic
	suite.True(true)
}

// Test Stop method
func (suite *JobManagerTestSuite) TestStop() {
	jm := NewJobManager()
	resultQueue := make(chan *types.JobResult, 10)

	jm.Start(suite.ctx, resultQueue)

	// Stop should close quit channel and wait for workers
	suite.NotPanics(func() {
		jm.Stop()
	})
}

// Test SubmitJob with valid job
func (suite *JobManagerTestSuite) TestSubmitJob() {
	jm := NewJobManager()

	// Test normal job submission
	job := &types.Job{
		ID:    1,
		URL:   "http://example.com",
		Path:  "data.value",
		Type:  types.Register,
		Nonce: 0,
		Delay: time.Second,
	}

	suite.NotPanics(func() {
		jm.SubmitJob(job)
	})

	// Job should be in queue
	suite.Equal(1, len(jm.jobQueue))
}

// Test SubmitJob with queue full
func (suite *JobManagerTestSuite) TestSubmitJob_QueueFull() {
	jm := NewJobManager()

	// Fill the queue to capacity
	queueSize := cap(jm.jobQueue)
	for i := 0; i < queueSize; i++ {
		job := &types.Job{
			ID:    uint64(i),
			URL:   "http://example.com",
			Path:  "data.value",
			Type:  types.Register,
			Nonce: 0,
			Delay: time.Second,
		}
		jm.SubmitJob(job)
	}

	// Queue should be full
	suite.Equal(queueSize, len(jm.jobQueue))

	// Additional job should be dropped (not panic)
	extraJob := &types.Job{
		ID:    999,
		URL:   "http://example.com",
		Path:  "data.value",
		Type:  types.Register,
		Nonce: 0,
		Delay: time.Second,
	}

	suite.NotPanics(func() {
		jm.SubmitJob(extraJob)
	})

	// Queue size should remain the same
	suite.Equal(queueSize, len(jm.jobQueue))
}

// Test SubmitJob with nil job
func (suite *JobManagerTestSuite) TestSubmitJob_NilJob() {
	jm := NewJobManager()

	suite.NotPanics(func() {
		jm.SubmitJob(nil)
	})

	// Should still have the nil job in queue
	suite.Equal(1, len(jm.jobQueue))
}

// Test worker Register job handling
func (suite *JobManagerTestSuite) TestWorker_RegisterJob() {
	jm := NewJobManager()
	resultQueue := make(chan *types.JobResult, 10)

	// Start workers
	jm.Start(suite.ctx, resultQueue)
	defer jm.Stop()

	job := &types.Job{
		ID:    1,
		URL:   "http://httpbin.org/json", // Use real URL that works
		Path:  "url",
		Type:  types.Register,
		Nonce: 0,
		Delay: time.Millisecond,
	}

	jm.SubmitJob(job)

	// Give worker time to process
	time.Sleep(100 * time.Millisecond)

	// Job should be in active jobs
	jm.activeJobsLock.Lock()
	_, exists := jm.activeJobs[job.ID]
	jm.activeJobsLock.Unlock()

	suite.True(exists)
}

// Test worker Update job handling
func (suite *JobManagerTestSuite) TestWorker_UpdateJob() {
	jm := NewJobManager()
	resultQueue := make(chan *types.JobResult, 10)

	// First add a register job
	registerJob := &types.Job{
		ID:    1,
		URL:   "http://httpbin.org/json",
		Path:  "url",
		Type:  types.Register,
		Nonce: 0,
		Delay: time.Millisecond,
	}

	jm.activeJobsLock.Lock()
	jm.activeJobs[registerJob.ID] = registerJob
	jm.activeJobsLock.Unlock()

	jm.Start(suite.ctx, resultQueue)
	defer jm.Stop()

	// Submit update job
	updateJob := &types.Job{
		ID:    1,
		URL:   "http://httpbin.org/json",
		Path:  "headers.Host",
		Type:  types.Update,
		Nonce: 1,
		Delay: time.Millisecond,
	}

	jm.SubmitJob(updateJob)

	// Give worker time to process
	time.Sleep(100 * time.Millisecond)

	// Job should be updated in active jobs
	jm.activeJobsLock.Lock()
	activeJob, exists := jm.activeJobs[updateJob.ID]
	jm.activeJobsLock.Unlock()

	suite.True(exists)
	suite.Equal("headers.Host", activeJob.Path)
}

// Test worker Complete job handling
func (suite *JobManagerTestSuite) TestWorker_CompleteJob() {
	jm := NewJobManager()
	resultQueue := make(chan *types.JobResult, 10)

	// First add a register job
	registerJob := &types.Job{
		ID:    1,
		URL:   "http://httpbin.org/json",
		Path:  "url",
		Type:  types.Register,
		Nonce: 0,
		Delay: time.Millisecond,
	}

	jm.activeJobsLock.Lock()
	jm.activeJobs[registerJob.ID] = registerJob
	jm.activeJobsLock.Unlock()

	jm.Start(suite.ctx, resultQueue)
	defer jm.Stop()

	// Submit complete job
	completeJob := &types.Job{
		ID:    1,
		Type:  types.Complete,
		Nonce: 2,
	}

	jm.SubmitJob(completeJob)

	// Give worker time to process
	time.Sleep(100 * time.Millisecond)

	// Job should still exist but with Complete type
	jm.activeJobsLock.Lock()
	activeJob, exists := jm.activeJobs[completeJob.ID]
	jm.activeJobsLock.Unlock()

	suite.True(exists)
	suite.Equal(types.Complete, activeJob.Type)
}

// Test worker Complete job with non-existent job
func (suite *JobManagerTestSuite) TestWorker_CompleteJob_NotExists() {
	jm := NewJobManager()
	resultQueue := make(chan *types.JobResult, 10)

	jm.Start(suite.ctx, resultQueue)
	defer jm.Stop()

	// Submit complete job for non-existent job
	completeJob := &types.Job{
		ID:    999,
		Type:  types.Complete,
		Nonce: 1,
	}

	jm.SubmitJob(completeJob)

	// Give worker time to process
	time.Sleep(50 * time.Millisecond)

	// Job should not be in active jobs
	jm.activeJobsLock.Lock()
	_, exists := jm.activeJobs[completeJob.ID]
	jm.activeJobsLock.Unlock()

	suite.False(exists)
}

// Test concurrent job submission
func (suite *JobManagerTestSuite) TestConcurrentJobSubmission() {
	jm := NewJobManager()

	var wg sync.WaitGroup
	numGoroutines := 10
	jobsPerGoroutine := 5

	// Start concurrent job submissions
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < jobsPerGoroutine; j++ {
				job := &types.Job{
					ID:    uint64(goroutineID*jobsPerGoroutine + j),
					URL:   "http://example.com",
					Path:  "data.value",
					Type:  types.Register,
					Nonce: 0,
					Delay: time.Millisecond,
				}
				jm.SubmitJob(job)
			}
		}(i)
	}

	wg.Wait()

	// Should not exceed queue capacity, some jobs may be dropped
	suite.LessOrEqual(len(jm.jobQueue), cap(jm.jobQueue))
}

// Test worker context cancellation
func (suite *JobManagerTestSuite) TestWorker_ContextCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	jm := NewJobManager()
	resultQueue := make(chan *types.JobResult, 10)

	jm.Start(ctx, resultQueue)

	// Cancel context
	cancel()

	// Stop should not hang
	done := make(chan bool, 1)
	go func() {
		jm.Stop()
		done <- true
	}()

	select {
	case <-done:
		suite.True(true) // Success
	case <-time.After(5 * time.Second):
		suite.Fail("Stop() timed out after context cancellation")
	}
}

// Test worker count matches CPU count
func (suite *JobManagerTestSuite) TestWorkerCount() {
	jm := NewJobManager()
	resultQueue := make(chan *types.JobResult, 10)

	// Count active goroutines before start
	initialGoroutines := runtime.NumGoroutine()

	jm.Start(suite.ctx, resultQueue)

	// Give time for workers to start
	time.Sleep(50 * time.Millisecond)

	currentGoroutines := runtime.NumGoroutine()
	newGoroutines := currentGoroutines - initialGoroutines

	// Should have created workers equal to CPU count
	// Note: This is approximate due to other goroutines that may start
	suite.GreaterOrEqual(newGoroutines, runtime.NumCPU())

	jm.Stop()
}

// Test edge cases
func (suite *JobManagerTestSuite) TestEdgeCases() {
	// Test Stop without Start
	suite.Run("stop without start", func() {
		jm := NewJobManager()
		suite.NotPanics(func() {
			jm.Stop()
		})
	})

	// Test multiple Start calls
	suite.Run("multiple starts", func() {
		jm := NewJobManager()
		resultQueue := make(chan *types.JobResult, 10)

		jm.Start(suite.ctx, resultQueue)

		// Second start should launch more workers
		suite.NotPanics(func() {
			jm.Start(suite.ctx, resultQueue)
		})

		jm.Stop()
	})
}

// Test performance with high load
func (suite *JobManagerTestSuite) TestHighLoadPerformance() {
	jm := NewJobManager()
	resultQueue := make(chan *types.JobResult, 1000)

	jm.Start(suite.ctx, resultQueue)
	defer jm.Stop()

	// Submit many jobs quickly
	numJobs := 50 // Reduced number to avoid HTTP timeouts
	start := time.Now()

	for i := 0; i < numJobs; i++ {
		job := &types.Job{
			ID:    uint64(i),
			URL:   "http://example.com",
			Path:  "data.value",
			Type:  types.Register,
			Nonce: 0,
			Delay: time.Microsecond,
		}
		jm.SubmitJob(job)
	}

	elapsed := time.Since(start)

	// Should complete submissions quickly (under 1 second)
	suite.Less(elapsed, time.Second)

	// Give time for processing
	time.Sleep(100 * time.Millisecond)

	// Test passes if no panic occurs
	suite.True(true)
}
