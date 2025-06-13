package woker

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/GPTx-global/guru/oracle/types"
)

type JobManager struct {
	activeJobs     map[uint64]*types.Job
	activeJobsLock sync.Mutex

	jobQueue chan *types.Job
	quit     chan struct{}
	wg       sync.WaitGroup
}

// NewJobManager creates a new job manager with CPU-based worker pool capacity
func NewJobManager() *JobManager {
	jm := &JobManager{
		activeJobs:     make(map[uint64]*types.Job),
		activeJobsLock: sync.Mutex{},
		jobQueue:       make(chan *types.Job, runtime.NumCPU()*4),
		quit:           make(chan struct{}),
		wg:             sync.WaitGroup{},
	}

	return jm
}

// Start launches worker goroutines equal to the number of CPU cores
func (jm *JobManager) Start(ctx context.Context, resultQueue chan<- *types.JobResult) {
	for i := 0; i < runtime.NumCPU(); i++ {
		jm.wg.Add(1)
		go jm.worker(ctx, resultQueue)
	}
}

// Stop gracefully shuts down all workers and waits for completion
func (jm *JobManager) Stop() {
	close(jm.quit)
	jm.wg.Wait()
}

// SubmitJob adds a new job to the processing queue, drops job if queue is full
func (jm *JobManager) SubmitJob(job *types.Job) {
	select {
	case jm.jobQueue <- job:
	default:
		fmt.Println("job queue is full, drop job")
		return
	}
}

// worker processes jobs from the queue and sends results to the result channel
func (jm *JobManager) worker(ctx context.Context, resultQueue chan<- *types.JobResult) {
	defer jm.wg.Done()

	for {
		select {
		case job := <-jm.jobQueue:
			jm.activeJobsLock.Lock()
			if _, ok := jm.activeJobs[job.ID]; !ok {
				jm.activeJobs[job.ID] = job
			} else {
				existingJob := jm.activeJobs[job.ID]
				reciveNonce, existNonce := job.Nonce, existingJob.Nonce
				job = existingJob
				if reciveNonce > existNonce {
					job.Nonce = reciveNonce
				}
			}
			jm.activeJobsLock.Unlock()
			job.Nonce++
			jr := executeJob(job)
			if jr != nil {
				resultQueue <- jr
			}
		case <-jm.quit:
			return
		case <-ctx.Done():
			return
		}
	}
}
