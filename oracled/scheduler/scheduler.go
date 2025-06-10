package scheduler

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/GPTx-global/guru/app"
	"github.com/GPTx-global/guru/encoding"
	"github.com/GPTx-global/guru/oracled/types"
	oracletypes "github.com/GPTx-global/guru/x/oracle/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type Scheduler struct {
	eventCh       <-chan coretypes.ResultEvent
	activeJobs    map[uint64]*types.Job // RequestID가 uint64이므로 key도 uint64로 변경
	activeJobsMux sync.Mutex
	resultCh      chan types.OracleData
	txDecoder     sdk.TxDecoder
	cdc           codec.Codec
}

func NewScheduler() *Scheduler {
	fmt.Printf("[ START ] NewScheduler\n")

	encodingConfig := encoding.MakeConfig(app.ModuleBasics)
	s := &Scheduler{
		eventCh:       nil,
		activeJobs:    make(map[uint64]*types.Job),
		activeJobsMux: sync.Mutex{},
		resultCh:      make(chan types.OracleData, 64),
		txDecoder:     encodingConfig.TxConfig.TxDecoder(),
		cdc:           encodingConfig.Codec,
	}

	fmt.Printf("[  END  ] NewScheduler: SUCCESS - resultCh buffer size=64\n")
	return s
}

func (s *Scheduler) Start(ctx context.Context) {
	fmt.Printf("[ START ] Start\n")

	// fmt.Printf("[INFO    ] Start: Starting eventProcessor goroutine\n")
	go s.eventProcessor(ctx)

	fmt.Printf("[  END  ] Start: SUCCESS\n")
}

func (s *Scheduler) eventProcessor(ctx context.Context) {
	fmt.Printf("[ START ] eventProcessor\n")

	for {
		select {
		case event := <-s.eventCh:
			fmt.Printf("[ EVENT ] eventProcessor: Received event\n")
			job, err := s.eventToJob(event)
			if err != nil || job == nil {
				fmt.Printf("[WARN    ] eventProcessor: Failed to parse oracle event: %v\n", err)
				continue
			}

			// fmt.Printf("[INFO    ] eventProcessor: Starting processJob for ID: %d\n", job.ID)
			go s.processJob(ctx, job)
		case <-ctx.Done():
			fmt.Printf("[  END  ] eventProcessor: SUCCESS - context cancelled\n")
			return
		}
	}
}

func (s *Scheduler) eventToJob(event coretypes.ResultEvent) (*types.Job, error) {
	fmt.Printf("[ START ] eventToJob\n")

	switch eventData := event.Data.(type) {
	case tmtypes.EventDataTx:
		// fmt.Printf("[INFO    ] eventToJob: Processing EventDataTx\n")
		// 트랜잭션 바이트 데이터를 디코딩
		tx, err := s.txDecoder(eventData.Tx)
		if err != nil {
			fmt.Printf("[  END  ] eventToJob: ERROR - failed to decode transaction: %v\n", err)
			return nil, fmt.Errorf("failed to decode transaction: %w", err)
		}

		// 트랜잭션 내의 모든 메시지를 확인
		msgs := tx.GetMsgs()
		// fmt.Printf("[INFO    ] eventToJob: Processing %d messages\n", len(msgs))

		for _, msg := range msgs {
			switch oracleMsg := msg.(type) {
			case *oracletypes.MsgRegisterOracleRequestDoc:
				// fmt.Printf("[INFO    ] eventToJob: Processing MsgRegisterOracleRequestDoc\n")
				// Register Oracle Request 메시지 처리 - 첫 번째 등록
				reqID, _ := strconv.ParseUint(event.Events["register_oracle_request_doc.request_id"][0], 10, 64)

				s.activeJobsMux.Lock()
				// 이미 존재하는 job인지 확인
				if _, exists := s.activeJobs[reqID]; exists {
					s.activeJobsMux.Unlock()
					fmt.Printf("[  END  ] eventToJob: ERROR - job already exists: %d\n", reqID)
					return nil, fmt.Errorf("job already exists: %d", reqID)
				}

				job := &types.Job{
					ID:          reqID,
					URL:         oracleMsg.RequestDoc.Endpoints[0].Url,
					Path:        oracleMsg.RequestDoc.Endpoints[0].ParseRule,
					Nonce:       1,
					Delay:       time.Duration(oracleMsg.RequestDoc.Period) * time.Second,
					MessageType: "register",
				}

				// activeJobs에 추가
				s.activeJobs[reqID] = job
				s.activeJobsMux.Unlock()

				fmt.Printf("[SUCCESS] eventToJob: Registered new job - ID: %d\n", job.ID)

				// 첫 번째 실행을 위해 job 반환
				fmt.Printf("[  END  ] eventToJob: SUCCESS - new job created\n")
				return job, nil

			case *oracletypes.MsgUpdateOracleRequestDoc:
				// fmt.Printf("[INFO    ] eventToJob: Processing MsgUpdateOracleRequestDoc\n")
				// Update Oracle Request 메시지 처리
				reqID := oracleMsg.RequestDoc.RequestId

				s.activeJobsMux.Lock()
				if existingJob, exists := s.activeJobs[reqID]; exists {
					// 기존 job 업데이트
					existingJob.URL = oracleMsg.RequestDoc.Endpoints[0].Url
					existingJob.Path = oracleMsg.RequestDoc.Endpoints[0].ParseRule
					existingJob.Delay = time.Duration(oracleMsg.RequestDoc.Period) * time.Second
					existingJob.MessageType = "update"
					s.activeJobsMux.Unlock()

					fmt.Printf("[SUCCESS] eventToJob: Updated existing job - ID: %d, URL: %s, Path: %s\n",
						existingJob.ID, existingJob.URL, existingJob.Path)
					fmt.Printf("[  END  ] eventToJob: SUCCESS - job updated\n")
					return existingJob, nil
				}
				s.activeJobsMux.Unlock()
				fmt.Printf("[  END  ] eventToJob: ERROR - job not found for update: %d\n", reqID)
				return nil, fmt.Errorf("job not found for update: %d", reqID)
			}
		}

	case tmtypes.EventDataNewBlock:
		// fmt.Printf("[INFO    ] eventToJob: Processing EventDataNewBlock\n")
		reqID, err := strconv.ParseUint(event.Events["complete_oracle_data_set.request_id"][0], 10, 64)
		if err != nil {
			fmt.Printf("[  END  ] eventToJob: ERROR - failed to parse request ID: %v\n", err)
			return nil, fmt.Errorf("failed to parse request ID: %w", err)
		}

		s.activeJobsMux.Lock()
		existingJob, exists := s.activeJobs[reqID]
		if !exists {
			s.activeJobsMux.Unlock()
			fmt.Printf("[  END  ] eventToJob: ERROR - job not found in activeJobs: %d\n", reqID)
			return nil, fmt.Errorf("job not found in activeJobs: %d", reqID)
		}

		// 기존 job의 nonce 증가
		oldNonce := existingJob.Nonce
		existingJob.Nonce++
		fmt.Printf("[SUCCESS] eventToJob: Incremented nonce for job ID=%d: %d -> %d\n",
			reqID, oldNonce, existingJob.Nonce)
		s.activeJobsMux.Unlock()

		// 업데이트된 job 반환
		fmt.Printf("[  END  ] eventToJob: SUCCESS - nonce incremented\n")
		return existingJob, nil

	default:
		fmt.Printf("[  END  ] eventToJob: ERROR - unsupported event data type: %T\n", event.Data)
		return nil, fmt.Errorf("unsupported event data type: %T", event.Data)
	}

	fmt.Printf("[  END  ] eventToJob: ERROR - no matching event type\n")
	return nil, nil
}

func (s *Scheduler) processJob(ctx context.Context, job *types.Job) {
	fmt.Printf("[ START ] processJob - ID: %d, Nonce: %d, Type: %s\n",
		job.ID, job.Nonce, job.MessageType)

	// MsgRegisterOracleRequestDoc의 경우 이미 activeJobs에 추가되었으므로
	// processJob에서는 실행만 진행
	// EventDataNewBlock의 경우도 nonce가 이미 증가되었으므로 실행만 진행

	// fmt.Printf("[INFO    ] processJob: Creating executor\n")
	executor := NewExecutor(ctx)

	// fmt.Printf("[INFO    ] processJob: Executing job\n")
	oracleData, err := executor.ExecuteJob(job)
	if err != nil {
		fmt.Printf("[  END  ] processJob: ERROR - failed to execute job %d: %v\n", job.ID, err)
		return
	}

	// fmt.Printf("[INFO    ] processJob: Job executed successfully, sending to resultCh\n")
	fmt.Printf("[SUCCESS] processJob: Oracle data created - RequestID: %d\n", oracleData.RequestID)
	s.resultCh <- *oracleData

	fmt.Printf("[  END  ] processJob: SUCCESS\n")
}

func (s *Scheduler) SetEventChannel(ch <-chan coretypes.ResultEvent) {
	// fmt.Printf("[INFO    ] SetEventChannel: Setting eventCh\n")
	s.eventCh = ch
}

func (s *Scheduler) GetResultChannel() <-chan types.OracleData {
	// fmt.Printf("[INFO    ] GetResultChannel: Returning resultCh\n")
	return s.resultCh
}
