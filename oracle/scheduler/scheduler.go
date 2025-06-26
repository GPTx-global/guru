package scheduler

import (
	"fmt"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"

	"github.com/GPTx-global/guru/oracle/config"
	"github.com/GPTx-global/guru/oracle/log"
	"github.com/GPTx-global/guru/oracle/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
)

// Scheduler manages job execution using a CPU-based worker pool architecture.
// It handles oracle request processing, job scheduling, and result collection.
type Scheduler struct {
	wg          sync.WaitGroup                        // WaitGroup for managing worker goroutines lifecycle
	quit        chan struct{}                         // Channel for signaling worker shutdown
	jobStore    cmap.ConcurrentMap[string, types.Job] // Thread-safe storage for active jobs
	jobQueue    chan types.Job                        // Channel for queuing jobs for execution
	resultQueue chan types.JobResult                  // Channel for collecting job execution results
}

// New creates a new Scheduler instance with initialized channels and storage.
// The scheduler starts with no active workers; call Start() to begin processing.
func New() *Scheduler {
	return &Scheduler{
		wg:          sync.WaitGroup{},
		quit:        make(chan struct{}),
		jobStore:    cmap.New[types.Job](),
		jobQueue:    make(chan types.Job, config.ChannelSize()),
		resultQueue: make(chan types.JobResult, config.ChannelSize()),
	}
}

// Start initializes and starts worker goroutines based on the number of CPU cores.
// Each worker runs independently and processes jobs from the job queue.
func (s *Scheduler) Start() {
	for i := 0; i < runtime.NumCPU(); i++ {
		s.wg.Add(1)
		go s.worker()
	}
}

// Stop gracefully shuts down all worker goroutines and waits for them to complete.
// After calling this method, no new jobs will be processed.
func (s *Scheduler) Stop() {
	close(s.quit)
	s.wg.Wait()
}

// ProcessRequestDoc processes an oracle request document and creates a corresponding job.
// It validates the request, checks if this oracle instance is responsible for it,
// and queues the job for execution if the status is enabled.
func (s *Scheduler) ProcessRequestDoc(requestDoc oracletypes.OracleRequestDoc) error {
	// Find the index of this oracle instance in the account list
	index := slices.Index(requestDoc.AccountList, config.Address().String())
	if index == -1 {
		return fmt.Errorf("request document not assigned to this oracle instance %s", config.Address().String())
	} else {
		// Calculate endpoint index based on account position to distribute load
		index = (index + 1) % len(requestDoc.Endpoints)
	}

	// Determine the current nonce for this job
	var currentNonce uint64
	if job, ok := s.jobStore.Get(strconv.FormatUint(requestDoc.RequestId, 10)); ok {
		currentNonce = job.Nonce
	} else {
		currentNonce = requestDoc.Nonce
	}

	// Create a new job from the request document
	job := types.Job{
		ID:     requestDoc.RequestId,
		URL:    requestDoc.Endpoints[index].Url,
		Path:   requestDoc.Endpoints[index].ParseRule,
		Nonce:  max(currentNonce, requestDoc.Nonce),
		Delay:  time.Duration(requestDoc.Period) * time.Second,
		Status: requestDoc.Status,
	}

	// Store the job for future reference
	s.jobStore.Set(strconv.FormatUint(requestDoc.RequestId, 10), job)

	// Only queue enabled jobs for execution
	if job.Status != oracletypes.RequestStatus_REQUEST_STATUS_ENABLED {
		return fmt.Errorf("job status is not enabled for job ID: %d", job.ID)
	}

	// Queue the job for execution
	s.jobQueue <- job

	return nil
}

// ProcessComplete handles completion events by updating job nonce and re-queuing for next execution.
// This method is called when oracle data submission is completed on the blockchain.
func (s *Scheduler) ProcessComplete(id string, nonce uint64) error {
	job, ok := s.jobStore.Get(id)
	if !ok {
		return fmt.Errorf("job not found with ID: %s", id)
	}

	// Update job nonce to the completed nonce
	job.Nonce = nonce

	// Re-queue the job for next execution cycle
	s.jobQueue <- job

	return nil
}

// Results returns a read-only channel for receiving job execution results.
// Consumers can range over this channel to process completed job results.
func (s *Scheduler) Results() <-chan types.JobResult {
	return s.resultQueue
}

// worker is the main worker goroutine that processes jobs from the job queue.
// It increments job nonce, executes the job, and sends results to the result queue.
func (s *Scheduler) worker() {
	defer s.wg.Done()

	for {
		select {
		case job := <-s.jobQueue:
			// Increment nonce for this execution
			job.Nonce++
			// Update the stored job with new nonce
			s.jobStore.Set(strconv.FormatUint(job.ID, 10), job)

			// Execute the job and get the result
			jobResult, err := executeJob(job)
			if err != nil {
				log.Errorf("failed to execute job ID %d: %v", job.ID, err)
				continue
			}

			// Send the result to the result queue
			s.resultQueue <- jobResult

		case <-s.quit:
			// Shutdown signal received, exit worker
			return
		}
	}
}
