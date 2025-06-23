package woker

import (
	"context"
	"runtime"
	"sync"

	"github.com/GPTx-global/guru/oracle/log"
	"github.com/GPTx-global/guru/oracle/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
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
		jobQueue:       make(chan *types.Job, 1<<9),
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
		log.Debugf("job queue is full, drop job")
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

			switch job.Type {
			case types.Register:
				log.Debugf("register job %d", job.ID)
				if _, ok := jm.activeJobs[job.ID]; !ok {
					jm.activeJobs[job.ID] = job
				}
			case types.Update:
				log.Debugf("update job %d", job.ID)
				jm.activeJobsLock.Unlock()
				// TODO: logic to update job
				// jm.activeJobs[job.ID] = job
				continue
			case types.Complete:
				log.Debugf("complete job %d", job.ID)
				if existingJob, ok := jm.activeJobs[job.ID]; ok {
					if existingJob.Status != oracletypes.RequestStatus_REQUEST_STATUS_ENABLED.String() {
						log.Debugf("job %d is not enabled", job.ID)
						jm.activeJobsLock.Unlock()
						continue
					}
					nonce := max(job.Nonce, existingJob.Nonce)
					job = existingJob
					job.Nonce = nonce
				} else {
					log.Debugf("job %d is not MINE!!", job.ID)
					jm.activeJobsLock.Unlock()
					continue
				}
				job.Type = types.Complete
			case types.GasPrice:
				log.Debugf("gas price job %s", job.GasPrice)
				jm.activeJobsLock.Unlock()
				continue
			}

			jm.activeJobsLock.Unlock()
			job.Nonce++
			jr, err := executeJob(job)
			if err != nil {
				log.Errorf("execute job %d failed: %v", job.ID, err)
				continue
			}
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
