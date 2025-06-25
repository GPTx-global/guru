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

type Scheduler struct {
	wg          sync.WaitGroup
	quit        chan struct{}
	jobStore    cmap.ConcurrentMap[string, types.Job]
	jobQueue    chan types.Job
	resultQueue chan types.JobResult
}

func New() *Scheduler {
	return &Scheduler{
		wg:          sync.WaitGroup{},
		quit:        make(chan struct{}),
		jobStore:    cmap.New[types.Job](),
		jobQueue:    make(chan types.Job, config.ChannelSize()),
		resultQueue: make(chan types.JobResult, config.ChannelSize()),
	}
}

func (s *Scheduler) Start() {
	for i := 0; i < runtime.NumCPU(); i++ {
		s.wg.Add(1)
		go s.worker()
	}
}

func (s *Scheduler) Stop() {
	close(s.quit)
	s.wg.Wait()
}

func (s *Scheduler) ProcessRequestDoc(requestDoc oracletypes.OracleRequestDoc) error {
	index := slices.Index(requestDoc.AccountList, config.Address().String())
	if index == -1 {
		return fmt.Errorf("request doc not for %s", config.Address().String())
	} else {
		index = (index + 1) % len(requestDoc.Endpoints)
	}

	var currentNonce uint64
	if job, ok := s.jobStore.Get(strconv.FormatUint(requestDoc.RequestId, 10)); ok {
		currentNonce = job.Nonce
	} else {
		currentNonce = requestDoc.Nonce
	}

	job := types.Job{
		ID:     requestDoc.RequestId,
		URL:    requestDoc.Endpoints[index].Url,
		Path:   requestDoc.Endpoints[index].ParseRule,
		Nonce:  max(currentNonce, requestDoc.Nonce),
		Delay:  time.Duration(requestDoc.Period) * time.Second,
		Status: requestDoc.Status,
	}

	s.jobStore.Set(strconv.FormatUint(requestDoc.RequestId, 10), job)

	if job.Status != oracletypes.RequestStatus_REQUEST_STATUS_ENABLED {
		return fmt.Errorf("job status is not enabled: %d", job.ID)
	}

	s.jobQueue <- job

	return nil
}

func (s *Scheduler) ProcessComplete(id string, nonce uint64) error {
	job, ok := s.jobStore.Get(id)
	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}

	job.Nonce = nonce

	s.jobQueue <- job

	return nil
}

func (s *Scheduler) Result() <-chan types.JobResult {
	return s.resultQueue
}

func (s *Scheduler) worker() {
	defer s.wg.Done()

	for {
		select {
		case job := <-s.jobQueue:
			job.Nonce++
			s.jobStore.Set(strconv.FormatUint(job.ID, 10), job)

			jr, err := executeJob(job)
			if err != nil {
				log.Errorf("failed to execute job: %v", err)
				continue
			}

			s.resultQueue <- jr

		case <-s.quit:
			return
		}
	}
}
