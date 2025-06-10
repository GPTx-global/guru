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
	encodingConfig := encoding.MakeConfig(app.ModuleBasics)
	return &Scheduler{
		eventCh:       nil,
		activeJobs:    make(map[uint64]*types.Job),
		activeJobsMux: sync.Mutex{},
		resultCh:      make(chan types.OracleData, 64),
		txDecoder:     encodingConfig.TxConfig.TxDecoder(),
		cdc:           encodingConfig.Codec,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	// test용 activeJobs 추가
	// s.activeJobs[0] = &types.Job{
	// 	ID:    0,
	// 	URL:   "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT",
	// 	Path:  "price",
	// 	Nonce: 0,
	// 	Delay: 0,
	// }

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

	switch eventData := event.Data.(type) {
	case tmtypes.EventDataTx:
		// 트랜잭션 바이트 데이터를 디코딩
		tx, err := s.txDecoder(eventData.Tx)
		if err != nil {
			return nil, fmt.Errorf("failed to decode transaction: %w", err)
		}

		// 트랜잭션 내의 모든 메시지를 확인
		msgs := tx.GetMsgs()
		for _, msg := range msgs {
			switch oracleMsg := msg.(type) {
			case *oracletypes.MsgRegisterOracleRequestDoc:
				// 이벤트로 값 가져오기
				// Register Oracle Request 메시지 처리
				// job.ID = oracleMsg.RequestDoc.RequestId
				reqID, _ := strconv.ParseUint(event.Events["register_oracle_request_doc.request_id"][0], 10, 64)
				job.ID = reqID
				job.URL = oracleMsg.RequestDoc.Endpoints[0].Url
				job.Path = oracleMsg.RequestDoc.Endpoints[0].ParseRule
				job.Nonce = 0
				job.Delay = time.Duration(oracleMsg.RequestDoc.Period) * time.Second
				job.MessageType = "register"

				fmt.Printf("Parsed MsgRegisterOracleRequestDoc: ID=%d, URL=%s, Path=%s, Period=%d, MessageType=%s\n",
					job.ID, job.URL, job.Path, oracleMsg.RequestDoc.Period, job.MessageType)
				return job, nil

			case *oracletypes.MsgUpdateOracleRequestDoc:
				// Update Oracle Request 메시지 처리
				job.ID = oracleMsg.RequestDoc.RequestId
				job.URL = oracleMsg.RequestDoc.Endpoints[0].Url
				job.Path = oracleMsg.RequestDoc.Endpoints[0].ParseRule
				job.Nonce = 0
				job.Delay = time.Duration(oracleMsg.RequestDoc.Period) * time.Second
				job.MessageType = "update"

				fmt.Printf("Parsed MsgUpdateOracleRequestDoc: ID=%d, URL=%s, Path=%s, Period=%d, MessageType=%s\n",
					job.ID, job.URL, job.Path, oracleMsg.RequestDoc.Period, job.MessageType)
				return job, nil
			}
		}

	case tmtypes.EventDataNewBlock:
		reqID, err := strconv.ParseUint(event.Events["complete_oracle_data_set.request_id"][0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse request ID: %w", err)
		}

		nonce, err := strconv.ParseUint(event.Events["complete_oracle_data_set.nonce"][0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse nonce: %w", err)
		}

		job := &types.Job{
			ID:    reqID,
			Nonce: nonce,
		}

		_, ok := s.activeJobs[job.ID]
		if ok {
			return job, nil
		} else {
			return nil, fmt.Errorf("job not found: %d", reqID)
		}

		// if oracleIds, exists := event.Events["alpha.OracleId"]; exists && len(oracleIds) > 0 {
		// 	if parsedID, err := strconv.ParseUint(oracleIds[0], 10, 64); err == nil {
		// 		job.ID = parsedID
		// 	}
		// 	job.URL = "https://api.binance.com/api/v3/ticker/price?symbol=BTCUSDT" // 기본값
		// 	job.Path = "price"
		// 	job.Nonce = 0
		// 	if periods, exists := event.Events["register_oracle_request_doc.period"]; exists && len(periods) > 0 {
		// 		if delay, err := strconv.Atoi(periods[0]); err == nil {
		// 			job.Delay = time.Duration(delay) * time.Second
		// 		}
		// 	}
		// }
	default:
		return nil, fmt.Errorf("unsupported event data type: %T", event.Data)
	}

	return job, nil
}

func (s *Scheduler) processJob(ctx context.Context, job *types.Job) {
	// job에 address list를 추가하고 해당 주소에 포함되는지 확인
	// if !strings.Contains(event.Events[prefix+".account_list"][0], c.txBuilder.clientCtx.GetFromAddress().String()) {}
	
	

	job.Nonce++
	fmt.Printf("Processing job: %d, Nonce: %d\n", job.ID, job.Nonce)
	s.activeJobsMux.Lock()

	// existingJob, ok := s.activeJobs[job.ID]
	// if !ok {
	// 	s.activeJobs[job.ID] = job
	// 	job.Delay = 0
	// } else {
	// 	// if 
	// 	existingJob.Nonce++
	// 	job = existingJob
	// }

	// if _, ok := s.activeJobs[job.ID]; !ok {
	// 	s.activeJobs[job.ID] = job
	// } else {
	// 	// 기존 job 업데이트: nonce 증가
	// 	existingJob.Nonce++
	// 	job = existingJob
	// }
	// s.activeJobs[job.ID] = job

	s.activeJobs[job.ID] = job

	s.activeJobsMux.Unlock()

	executor := NewExecutor(ctx)
	oracleData, err := executor.ExecuteJob(job)
	if err != nil {
		fmt.Printf("Failed to execute job %d: %v\n", job.ID, err)
		return
	}
	fmt.Printf("Oracle data: %d\n", oracleData.RequestID)
	s.resultCh <- *oracleData
}

func (s *Scheduler) SetEventChannel(ch <-chan coretypes.ResultEvent) {
	s.eventCh = ch
}

func (s *Scheduler) GetResultChannel() <-chan types.OracleData {
	return s.resultCh
}
