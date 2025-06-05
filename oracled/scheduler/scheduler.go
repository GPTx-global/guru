package scheduler

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/GPTx-global/guru/oracled/types"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type Scheduler struct {
	eventCh       <-chan coretypes.ResultEvent
	activeJobs    map[string]*types.Job
	activeJobsMux sync.Mutex
	resultCh      chan types.OracleData
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		eventCh:       nil,
		activeJobs:    make(map[string]*types.Job),
		activeJobsMux: sync.Mutex{},
		resultCh:      make(chan types.OracleData, 64),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	// test용 activeJobs 추가
	s.activeJobs["0000"] = &types.Job{
		ID:    "0000",
		URL:   "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT",
		Path:  "price",
		Nonce: 0,
		Delay: 0,
	}

	go s.eventProcessor(ctx)
}

func (s *Scheduler) eventProcessor(ctx context.Context) {
	for {
		select {
		case event := <-s.eventCh:
			job, err := s.eventToJob(event)
			if err != nil || job == nil {
				fmt.Printf("Failed to parse oracle event: %v\n", err)
				continue
			}

			go s.processJob(ctx, job)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) eventToJob(event coretypes.ResultEvent) (*types.Job, error) {
	job := new(types.Job)

	switch event.Data.(type) {
	case tmtypes.EventDataNewBlock:
		// TODO: nonce 비교 확인 해야 함
		id := event.Events["alpha.OracleId"][0]
		s.activeJobsMux.Lock()
		job = s.activeJobs[id]
		s.activeJobsMux.Unlock()
	case tmtypes.EventDataTx:
		// TODO: register만 고려하고 있음, update 추가 해야 함
		// job.ID = event.Events["register_oracle_request_doc.request_id"][0]
		job.ID = event.Events["alpha.OracleId"][0]
		job.URL = "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT"
		job.Path = "price"
		job.Nonce = 0
		delay, _ := strconv.Atoi(event.Events["register_oracle_request_doc.period"][0])
		job.Delay = time.Duration(delay) * time.Second
	default:
		job = nil
	}

	return job, nil
}

func (s *Scheduler) processJob(ctx context.Context, job *types.Job) {
	fmt.Printf("Processing job: %s\n", job.ID)
	s.activeJobsMux.Lock()
	if existingJob, exists := s.activeJobs[job.ID]; exists {
		job = existingJob
		job.Nonce++
	}
	s.activeJobs[job.ID] = job
	s.activeJobsMux.Unlock()

	executor := NewExecutor(ctx)
	oracleData, err := executor.ExecuteJob(job)
	if err != nil {
		fmt.Printf("Failed to execute job %s: %v\n", job.ID, err)
		return
	}
	fmt.Printf("Oracle data: %s\n", oracleData.RequestID)
	s.resultCh <- *oracleData
}

func (s *Scheduler) SetEventChannel(ch <-chan coretypes.ResultEvent) {
	s.eventCh = ch
}

func (s *Scheduler) GetResultChannel() <-chan types.OracleData {
	return s.resultCh
}
