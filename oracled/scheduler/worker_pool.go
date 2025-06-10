package scheduler

import (
	"context"
	"fmt"
	"sync"

	"github.com/GPTx-global/guru/oracled/types"
)

// JobWorkerPool job 처리를 위한 워커 풀
type JobWorkerPool struct {
	maxWorkers  int
	jobQueue    chan *types.Job
	workerQueue chan chan *types.Job
	quit        chan bool
	wg          sync.WaitGroup
	scheduler   *Scheduler
}

// NewJobWorkerPool 새로운 워커 풀 생성
func NewJobWorkerPool(maxWorkers int, scheduler *Scheduler) *JobWorkerPool {
	fmt.Printf("[ START ] NewJobWorkerPool - MaxWorkers: %d\n", maxWorkers)

	pool := &JobWorkerPool{
		maxWorkers:  maxWorkers,
		jobQueue:    make(chan *types.Job, maxWorkers*2), // 버퍼링된 job 큐
		workerQueue: make(chan chan *types.Job, maxWorkers),
		quit:        make(chan bool),
		scheduler:   scheduler,
	}

	fmt.Printf("[  END  ] NewJobWorkerPool: SUCCESS - buffer size: %d\n", maxWorkers*2)
	return pool
}

// Start 워커 풀 시작
func (p *JobWorkerPool) Start(ctx context.Context) {
	fmt.Printf("[ START ] JobWorkerPool.Start - Workers: %d\n", p.maxWorkers)

	// 워커들 시작
	for i := 0; i < p.maxWorkers; i++ {
		worker := NewJobWorker(i, p.workerQueue, p.scheduler)
		worker.Start(ctx)
	}

	// 디스패처 시작
	go p.dispatcher(ctx)

	fmt.Printf("[  END  ] JobWorkerPool.Start: SUCCESS\n")
}

// Stop 워커 풀 중지
func (p *JobWorkerPool) Stop() {
	fmt.Printf("[ START ] JobWorkerPool.Stop\n")

	close(p.quit)
	p.wg.Wait()

	fmt.Printf("[  END  ] JobWorkerPool.Stop: SUCCESS\n")
}

// SubmitJob job을 워커 풀에 제출
func (p *JobWorkerPool) SubmitJob(job *types.Job) bool {
	select {
	case p.jobQueue <- job:
		return true
	default:
		fmt.Printf("[  WARN ] JobWorkerPool: Job queue full, dropping job %d\n", job.ID)
		return false
	}
}

// dispatcher job을 워커에게 분배
func (p *JobWorkerPool) dispatcher(ctx context.Context) {
	fmt.Printf("[ START ] JobWorkerPool.dispatcher\n")

	for {
		select {
		case job := <-p.jobQueue:
			// 사용 가능한 워커 찾기
			select {
			case workerJobQueue := <-p.workerQueue:
				// 워커에게 job 전달
				workerJobQueue <- job
			case <-ctx.Done():
				fmt.Printf("[  END  ] JobWorkerPool.dispatcher: CANCELLED\n")
				return
			}
		case <-p.quit:
			fmt.Printf("[  END  ] JobWorkerPool.dispatcher: SUCCESS - quit signal\n")
			return
		case <-ctx.Done():
			fmt.Printf("[  END  ] JobWorkerPool.dispatcher: SUCCESS - context cancelled\n")
			return
		}
	}
}

// JobWorker 개별 워커
type JobWorker struct {
	id          int
	jobQueue    chan *types.Job
	workerQueue chan chan *types.Job
	quit        chan bool
	scheduler   *Scheduler
}

// NewJobWorker 새로운 워커 생성
func NewJobWorker(id int, workerQueue chan chan *types.Job, scheduler *Scheduler) *JobWorker {
	return &JobWorker{
		id:          id,
		jobQueue:    make(chan *types.Job),
		workerQueue: workerQueue,
		quit:        make(chan bool),
		scheduler:   scheduler,
	}
}

// Start 워커 시작
func (w *JobWorker) Start(ctx context.Context) {
	go func() {
		fmt.Printf("[ START ] JobWorker-%d\n", w.id)

		for {
			// 워커 큐에 자신의 job 채널 등록
			w.workerQueue <- w.jobQueue

			select {
			case job := <-w.jobQueue:
				// job 처리
				fmt.Printf("[ WORKER] JobWorker-%d processing job %d\n", w.id, job.ID)
				w.scheduler.processJobWithRetry(ctx, job)

			case <-w.quit:
				fmt.Printf("[  END  ] JobWorker-%d: SUCCESS - quit signal\n", w.id)
				return
			case <-ctx.Done():
				fmt.Printf("[  END  ] JobWorker-%d: SUCCESS - context cancelled\n", w.id)
				return
			}
		}
	}()
}

// Stop 워커 중지
func (w *JobWorker) Stop() {
	w.quit <- true
}
