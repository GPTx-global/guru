package woker

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/GPTx-global/guru/oracle/types"
)

type JobManager struct {
	jobQueue chan *types.Job
	quit     chan struct{}
	wg       sync.WaitGroup
}

func NewJobManager() *JobManager {
	jm := new(JobManager)
	jm.jobQueue = make(chan *types.Job, runtime.NumCPU()*4)
	jm.quit = make(chan struct{})
	jm.wg = sync.WaitGroup{}

	return jm
}

func (jm *JobManager) Start(ctx context.Context) {
	for i := 0; i < runtime.NumCPU(); i++ {
		jm.wg.Add(1)
		go jm.worker(ctx)
	}
}

func (jm *JobManager) Stop() {
	close(jm.quit)
	jm.wg.Wait()
}

func (jm *JobManager) SubmitJob(job *types.Job) {
	select {
	case jm.jobQueue <- job:
	default:
		fmt.Println("job queue is full, drop job")
		return
	}
}

func (jm *JobManager) worker(ctx context.Context) {
	defer jm.wg.Done()

	for {
		select {
		case job := <-jm.jobQueue:
			jr := executeJob(job)
			if jr != nil {

			}
		case <-jm.quit:
			return
		case <-ctx.Done():
			return
		}
	}
}
